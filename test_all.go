//go:build ignore

// Tests live du SDK Kappelas Go.
//
// Ce fichier teste :
//   - Connexion WebSocket
//   - Profil bot
//   - Chats : list (avec offset), Iterate, GetMyGroups
//   - Messages : texte, formatage riche, photo, vidéo, document, audio, carousel
//   - Keyboards : inline, reply, scroll (formes courte + longue + mixte)
//   - reply_to_id (citation)
//   - delete_previous
//   - Edit texte + Edit clavier seul
//   - Delete
//   - Typing indicator
//   - Webhooks : GetInfo, handleWebhook (unit test sans serveur HTTP)
//   - Membres du groupe : GetAdministrators, GetMember, getMember NOT_FOUND
//   - Invite links (admin requis — skippé si pas de groupe admin)
//   - Messages dans le groupe : texte, reply_to_id, photo, carousel
//   - Gestion erreurs attendues : FORBIDDEN, KappelaError fields
//   - bot.Reply() : depuis *Message et *CallbackQuery
//
// Usage :
//
//	go run test_all.go <TOKEN> [CHAT_ID]
//
// ou via variables d'environnement :
//
//	KAPPELA_TOKEN=xxx CHAT_ID=130 go run test_all.go
package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	kappelas "github.com/Arnel7/kappelas-sdk-go"
)

// ─── Fichiers de test en mémoire ─────────────────────────────────────────────

// PNG 1×1 pixel transparent valide
var pngBytes, _ = base64.StdEncoding.DecodeString(
	"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
)

// WAV silence PCM 16bit 44100Hz mono
var wavBytes, _ = hex.DecodeString(
	"52494646" + "26000000" + "57415645" +
		"666d7420" + "10000000" + "01000100" + "44ac0000" + "88580100" + "02001000" +
		"64617461" + "02000000" + "0000",
)

// PDF minimal valide
var pdfBytes = []byte(
	"%PDF-1.0\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj " +
		"2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj " +
		"3 0 obj<</Type/Page/MediaBox[0 0 3 3]>>endobj\n" +
		"xref\n0 4\n0000000000 65535 f\n0000000009 00000 n\n" +
		"0000000058 00000 n\n0000000115 00000 n\n" +
		"trailer<</Size 4/Root 1 0 R>>\nstartxref\n190\n%%EOF",
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

var (
	passed  int
	failed  int
	skipped int
	ctx     = context.Background()
)

func section(title string) {
	sep := strings.Repeat("─", 60)
	fmt.Printf("\n%s\n  %s\n%s\n", sep, title, sep)
}

func run(label string, fn func() (any, error)) any {
	fmt.Printf("\n→ %s\n", label)
	result, err := fn()
	if err != nil {
		var ke *kappelas.KappelaError
		if errors.As(err, &ke) {
			fmt.Printf("  [✗] FAIL  KappelaError %s (%d): %s\n", ke.Code, ke.Status, ke.Message)
		} else {
			fmt.Printf("  [✗] FAIL  %v\n", err)
		}
		failed++
		return nil
	}
	b, _ := json.Marshal(result)
	out := string(b)
	if len(out) > 200 {
		out = out[:200] + "…"
	}
	fmt.Printf("  [✓] OK  %s\n", out)
	passed++
	return result
}

// runExpectError vérifie qu'une erreur avec le code attendu est bien retournée.
func runExpectError(label string, expectedCode kappelas.ErrorCode, fn func() (any, error)) bool {
	fmt.Printf("\n→ %s (attendu : %s)\n", label, expectedCode)
	_, err := fn()
	if err == nil {
		fmt.Printf("  [✗] FAIL — aurait dû retourner une erreur\n")
		failed++
		return false
	}
	var ke *kappelas.KappelaError
	if errors.As(err, &ke) && ke.Code == expectedCode {
		fmt.Printf("  [✓] OK  KappelaError %s reçue comme attendu\n", ke.Code)
		passed++
		return true
	}
	fmt.Printf("  [✗] FAIL — mauvaise erreur : %v\n", err)
	failed++
	return false
}

func skipTest(label, reason string) {
	fmt.Printf("\n→ %s\n  [⊘] SKIPPED — %s\n", label, reason)
	skipped++
}

func ptr(s string) *string { return &s }

func ptrStr(s *string, fallback string) string {
	if s != nil {
		return *s
	}
	return fallback
}

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	token := os.Getenv("KAPPELA_TOKEN")
	if token == "" && len(os.Args) > 1 {
		token = os.Args[1]
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "Usage: go run test_all.go <TOKEN> [CHAT_ID]")
		os.Exit(1)
	}

	chatID := int64(130)
	if v := os.Getenv("CHAT_ID"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n != 0 {
			chatID = n
		}
	} else if len(os.Args) > 2 {
		if n, err := strconv.ParseInt(os.Args[2], 10, 64); err == nil && n != 0 {
			chatID = n
		}
	}

	bot := kappelas.NewBot(token)

	// Attendre la connexion WebSocket
	var once sync.Once
	connected := make(chan struct{})
	bot.OnConnected(func() {
		once.Do(func() { close(connected) })
	})
	bot.OnError(func(err error) {
		fmt.Printf("[!] Erreur WS : %v\n", err)
	})
	bot.OnMessage(func(msg *kappelas.Message) {
		fmt.Printf("\n[→] Message reçu — chat_id=%d type=%s\n", msg.ChatID, msg.Type)
	})
	bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) {
		fmt.Printf("\n[→] Bouton cliqué — chat_id=%d data=%q\n", cb.ChatID, cb.CallbackData)
		bot.Reply(ctx, cb, "Tu as cliqué : "+cb.CallbackData)
	})

	bot.Start()

	select {
	case <-connected:
		fmt.Printf("[✓] Connecté — chat_id cible : %d\n", chatID)
	case <-time.After(10 * time.Second):
		fmt.Println("[✗] Timeout connexion WebSocket")
		os.Exit(1)
	}

	// ═══════════════════════════════════════════════════════════════════════════
	section("1. PROFIL")
	// ═══════════════════════════════════════════════════════════════════════════

	run("profile.Get()", func() (any, error) {
		return bot.Profile.Get(ctx)
	})

	// ═══════════════════════════════════════════════════════════════════════════
	section("2. CHATS")
	// ═══════════════════════════════════════════════════════════════════════════

	run("chats.List({ limit: 5 })", func() (any, error) {
		return bot.Chats.List(ctx, kappelas.GetChatsParams{Limit: 5})
	})

	run("chats.List() — avec offset", func() (any, error) {
		return bot.Chats.List(ctx, kappelas.GetChatsParams{Limit: 3, Offset: 1})
	})

	run("chats.Iterate() — premier chat", func() (any, error) {
		var first *kappelas.Chat
		err := bot.Chats.Iterate(ctx, 1, func(chat *kappelas.Chat) bool {
			first = chat
			return false // stop après le premier
		})
		if err != nil {
			return nil, err
		}
		if first == nil {
			return nil, fmt.Errorf("aucun chat retourné")
		}
		return map[string]any{"chat_id": first.ChatID, "type": first.Type}, nil
	})

	// GetMyGroups — récupéré ici pour les sections suivantes
	var myGroups *kappelas.GetMyGroupsResult
	if r, ok := run("chats.GetMyGroups()", func() (any, error) {
		return bot.Chats.GetMyGroups(ctx)
	}).(*kappelas.GetMyGroupsResult); ok && r != nil {
		myGroups = r
		fmt.Printf("  [ℹ] %d groupe(s)/canal\n", len(myGroups.Groups))
		for _, g := range myGroups.Groups {
			fmt.Printf("      %d (%s) %q → %s\n", g.ChatID, g.Type, ptrStr(g.Title, "(sans titre)"), g.BotRole)
		}
	}

	// Trouver un groupe quelconque et un groupe admin
	var anyGroupID, adminGroupID int64
	if myGroups != nil {
		for _, g := range myGroups.Groups {
			if anyGroupID == 0 {
				anyGroupID = g.ChatID
			}
			if g.BotRole == kappelas.ParticipantRoleAdmin && adminGroupID == 0 {
				adminGroupID = g.ChatID
			}
		}
	}
	if adminGroupID != 0 {
		fmt.Printf("  [ℹ] Groupe admin : chat_id=%d\n", adminGroupID)
	} else {
		fmt.Printf("  [ℹ] Aucun groupe admin — certains tests seront ignorés\n")
	}

	// ═══════════════════════════════════════════════════════════════════════════
	section("3. TEXTE SIMPLE + FORMATAGE")
	// ═══════════════════════════════════════════════════════════════════════════

	sentPlain := run("messages.Send() — texte simple", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "👋 Test SDK Go — texte simple",
		})
	})

	run("messages.Send() — gras, italique, barré, code inline", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "*gras*  __italique__  ~barré~  `code inline`",
		})
	})

	run("messages.Send() — bloc code", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Ta clé API :\n```\nsk_live_test_abc123xyz\n```",
		})
	})

	run("messages.Send() — citation (>)", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "> Question originale de l'utilisateur\n\nVoici la réponse détaillée.",
		})
	})

	run("messages.Send() — mention + commande", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Merci @test ! Tape /help pour voir les commandes disponibles.",
		})
	})

	run("messages.Send() — lien auto-détecté", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Visitez kappelas.com ou https://kappelas.com/docs",
		})
	})

	run("messages.Send() — formatage combiné", func() (any, error) {
		lines := []string{
			"🛒 *Récapitulatif commande*",
			"",
			"> Widget A × 2",
			"",
			"Total : **49 980 FCFA**",
			"Statut : `CONFIRMÉ`",
			"",
			"Questions ? contact@example.com ou /help",
		}
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   strings.Join(lines, "\n"),
		})
	})

	// ═══════════════════════════════════════════════════════════════════════════
	section("4. REPLY_TO_ID (CITATION DE MESSAGE)")
	// ═══════════════════════════════════════════════════════════════════════════

	if r, ok := sentPlain.(*kappelas.SendResult); ok && r != nil {
		run("messages.Send() — reply_to_id cite le message précédent", func() (any, error) {
			return bot.Messages.Send(ctx, kappelas.SendMessageParams{
				ChatID:    chatID,
				Text:      "↩️ Réponse avec citation du message précédent",
				ReplyToID: &r.MessageID,
			})
		})
	} else {
		skipTest("messages.Send() — reply_to_id", "message de référence absent")
	}

	// ═══════════════════════════════════════════════════════════════════════════
	section("4B. BOT.REPLY()")
	// ═══════════════════════════════════════════════════════════════════════════

	if r, ok := sentPlain.(*kappelas.SendResult); ok && r != nil {
		synthetic := &kappelas.Message{ID: r.MessageID, ChatID: chatID}

		run("bot.Reply(msg, text) — cite le message automatiquement", func() (any, error) {
			return bot.Reply(ctx, synthetic, "↩️ bot.Reply(msg) — reply_to_id injecté automatiquement")
		})

		run("bot.Reply(msg, text, opts) — avec inline keyboard", func() (any, error) {
			return bot.Reply(ctx, synthetic, "bot.Reply() avec clavier inline :", kappelas.SendMessageParams{
				ReplyMarkup: kappelas.InlineKeyboard{
					InlineKeyboard: [][]kappelas.InlineKeyboardButton{{
						{Text: "✅ OK", CallbackData: ptr("reply_ok")},
						{Text: "❌ Annuler", CallbackData: ptr("reply_cancel")},
					}},
				},
			})
		})
	} else {
		skipTest("bot.Reply(msg)", "message de référence absent")
	}

	// CallbackQuery synthétique — Reply sans quote banner
	cb := &kappelas.CallbackQuery{
		ChatID:       chatID,
		SenderID:     "test-uuid",
		CallbackData: "cb_test",
	}
	run("bot.Reply(cb, text) — callback_query, sans reply_to_id", func() (any, error) {
		return bot.Reply(ctx, cb, "↩️ bot.Reply(cb) — envoyé sans citation")
	})

	// ═══════════════════════════════════════════════════════════════════════════
	section("5. KEYBOARDS")
	// ═══════════════════════════════════════════════════════════════════════════

	// Inline keyboard
	run("messages.Send() — inline keyboard", func() (any, error) {
		url := "https://kappelas.com"
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Test inline keyboard :",
			ReplyMarkup: kappelas.InlineKeyboard{
				InlineKeyboard: [][]kappelas.InlineKeyboardButton{
					{
						{Text: "✅ Oui", CallbackData: ptr("yes")},
						{Text: "❌ Non", CallbackData: ptr("no")},
					},
					{
						{Text: "🌐 Site", URL: &url},
					},
				},
			},
		})
	})

	// Reply keyboard — forme courte (label == callback)
	run("messages.Send() — reply keyboard (forme courte)", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Test reply keyboard (forme courte) :",
			ReplyMarkup: kappelas.ReplyKeyboard{
				Keyboard: [][]kappelas.ReplyKeyboardButton{
					{{Text: "📦 Mes commandes"}, {Text: "❓ Aide"}},
					{{Text: "🔙 Retour"}},
				},
			},
		})
	})

	// Reply keyboard — forme longue (label ≠ callback)
	run("messages.Send() — reply keyboard (forme longue)", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Test reply keyboard (forme longue) :",
			ReplyMarkup: kappelas.ReplyKeyboard{
				Keyboard: [][]kappelas.ReplyKeyboardButton{
					{
						{Text: "✅ Oui", CallbackData: "confirm_yes"},
						{Text: "❌ Non", CallbackData: "confirm_no"},
					},
					{
						{Text: "↩ Annuler", CallbackData: "cancel"},
					},
				},
			},
		})
	})

	// Reply keyboard — forme mixte (short + long dans la même grille)
	run("messages.Send() — reply keyboard (mixte)", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Test reply keyboard (mixte) :",
			ReplyMarkup: kappelas.ReplyKeyboard{
				Keyboard: [][]kappelas.ReplyKeyboardButton{
					{
						{Text: "✅ Confirmer", CallbackData: "confirm"},
						{Text: "❓ Aide"}, // forme courte
					},
				},
			},
		})
	})

	// Scroll keyboard — forme courte
	run("messages.Send() — scroll keyboard (forme courte)", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Test scroll keyboard (forme courte) :",
			ReplyMarkup: kappelas.ScrollKeyboard{
				ScrollKeyboard: []kappelas.ScrollKeyboardButton{
					{Text: "📦 Commandes"}, {Text: "❓ Aide"}, {Text: "⚙️ Paramètres"},
				},
			},
		})
	})

	// Scroll keyboard — forme longue
	run("messages.Send() — scroll keyboard (forme longue)", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Test scroll keyboard (forme longue) :",
			ReplyMarkup: kappelas.ScrollKeyboard{
				ScrollKeyboard: []kappelas.ScrollKeyboardButton{
					{Text: "📦 Commandes", CallbackData: "menu_orders"},
					{Text: "❓ Aide", CallbackData: "menu_help"},
					{Text: "⚙️ Paramètres", CallbackData: "menu_settings"},
				},
			},
		})
	})

	// Scroll keyboard — forme mixte
	run("messages.Send() — scroll keyboard (mixte)", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Test scroll keyboard (mixte) :",
			ReplyMarkup: kappelas.ScrollKeyboard{
				ScrollKeyboard: []kappelas.ScrollKeyboardButton{
					{Text: "📦 Commandes", CallbackData: "menu_orders"},
					{Text: "❓ Aide"}, // forme courte
				},
			},
		})
	})

	// ═══════════════════════════════════════════════════════════════════════════
	section("6. MÉDIAS")
	// ═══════════════════════════════════════════════════════════════════════════

	run("messages.SendTyping() — show", func() (any, error) {
		return bot.Messages.SendTyping(ctx, kappelas.SendTypingParams{ChatID: chatID})
	})
	hide := false
	run("messages.SendTyping() — hide", func() (any, error) {
		return bot.Messages.SendTyping(ctx, kappelas.SendTypingParams{ChatID: chatID, IsTyping: &hide})
	})

	run("messages.SendPhoto()", func() (any, error) {
		return bot.Messages.SendPhoto(ctx, kappelas.SendMediaParams{
			ChatID:  chatID,
			File:    kappelas.FileInput{Data: pngBytes, Filename: "test.png", ContentType: "image/png"},
			Caption: "🖼 Photo test depuis le SDK Go",
		})
	})

	run("messages.SendDocument()", func() (any, error) {
		return bot.Messages.SendDocument(ctx, kappelas.SendMediaParams{
			ChatID:  chatID,
			File:    kappelas.FileInput{Data: pdfBytes, Filename: "test.pdf", ContentType: "application/pdf"},
			Caption: "📄 Document PDF test",
		})
	})

	run("messages.SendAudio()", func() (any, error) {
		return bot.Messages.SendAudio(ctx, kappelas.SendMediaParams{
			ChatID:  chatID,
			File:    kappelas.FileInput{Data: wavBytes, Filename: "test.wav", ContentType: "audio/wav"},
			Caption: "🔊 Audio test",
		})
	})

	run("messages.SendVideo()", func() (any, error) {
		// PNG utilisé comme placeholder (le serveur accepte la payload)
		return bot.Messages.SendVideo(ctx, kappelas.SendMediaParams{
			ChatID:  chatID,
			File:    kappelas.FileInput{Data: pngBytes, Filename: "test.mp4", ContentType: "video/mp4"},
			Caption: "🎬 Vidéo test (PNG placeholder)",
		})
	})

	// Photo avec reply_to_id
	if r, ok := sentPlain.(*kappelas.SendResult); ok && r != nil {
		run("messages.SendPhoto() — avec reply_to_id", func() (any, error) {
			return bot.Messages.SendPhoto(ctx, kappelas.SendMediaParams{
				ChatID:    chatID,
				File:      kappelas.FileInput{Data: pngBytes, Filename: "test.png", ContentType: "image/png"},
				Caption:   "🖼 Photo en réponse",
				ReplyToID: &r.MessageID,
			})
		})
	}

	// ═══════════════════════════════════════════════════════════════════════════
	section("7. CAROUSEL")
	// ═══════════════════════════════════════════════════════════════════════════

	run("messages.SendCarousel() — quick_reply forme courte", func() (any, error) {
		return bot.Messages.SendCarousel(ctx, kappelas.SendCarouselParams{
			ChatID: chatID,
			Text:   "🛍 Nos produits :",
			Carousel: []kappelas.CarouselCard{
				{ID: "p1", Title: "Widget A", Subtitle: ptr("9 990 FCFA"), ButtonText: ptr("Acheter")},
				{ID: "p2", Title: "Widget B", Subtitle: ptr("19 990 FCFA"), ButtonText: ptr("Acheter")},
			},
			QuickReplyButtons: []kappelas.ScrollKeyboardButton{
				{Text: "Voir plus"}, {Text: "Annuler"},
			},
		})
	})

	run("messages.SendCarousel() — quick_reply forme longue {text, callback_data}", func() (any, error) {
		return bot.Messages.SendCarousel(ctx, kappelas.SendCarouselParams{
			ChatID: chatID,
			Text:   "🛍 Sélection :",
			Carousel: []kappelas.CarouselCard{
				{ID: "p3", Title: "Widget C", Subtitle: ptr("4 990 FCFA"), ButtonText: ptr("Commander")},
			},
			QuickReplyButtons: []kappelas.ScrollKeyboardButton{
				{Text: "✅ Confirmer", CallbackData: "confirm"},
				{Text: "❌ Annuler", CallbackData: "cancel"},
			},
		})
	})

	if r, ok := sentPlain.(*kappelas.SendResult); ok && r != nil {
		run("messages.SendCarousel() — avec reply_to_id", func() (any, error) {
			return bot.Messages.SendCarousel(ctx, kappelas.SendCarouselParams{
				ChatID: chatID,
				Text:   "↩️ Voici les produits en lien avec ta question :",
				Carousel: []kappelas.CarouselCard{
					{ID: "p4", Title: "Offre spéciale", Subtitle: ptr("2 990 FCFA"), ButtonText: ptr("Commander")},
				},
				ReplyToID: &r.MessageID,
			})
		})
	}

	// ═══════════════════════════════════════════════════════════════════════════
	section("8. EDIT / DELETE")
	// ═══════════════════════════════════════════════════════════════════════════

	toEdit := run("messages.Send() — message à éditer", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "📝 Message original (sera modifié)",
			ReplyMarkup: kappelas.InlineKeyboard{
				InlineKeyboard: [][]kappelas.InlineKeyboardButton{{
					{Text: "🔴 Avant", CallbackData: ptr("before")},
				}},
			},
		})
	})

	if r, ok := toEdit.(*kappelas.SendResult); ok && r != nil {
		run("messages.Edit() — nouveau texte", func() (any, error) {
			return bot.Messages.Edit(ctx, kappelas.EditMessageParams{
				ChatID:    chatID,
				MessageID: r.MessageID,
				NewText:   "✅ Message modifié avec succès",
			})
		})

		done := "after"
		kb, _ := json.Marshal(kappelas.InlineKeyboard{
			InlineKeyboard: [][]kappelas.InlineKeyboardButton{
				{{Text: "🟢 Après", CallbackData: &done}},
			},
		})
		run("messages.Edit() — clavier inline seul (sans changer le texte)", func() (any, error) {
			return bot.Messages.Edit(ctx, kappelas.EditMessageParams{
				ChatID:       chatID,
				MessageID:    r.MessageID,
				NewExtraData: kb,
			})
		})
	}

	// Supprimer le message texte du début
	if r, ok := sentPlain.(*kappelas.SendResult); ok && r != nil {
		run(fmt.Sprintf("messages.Delete() — message_id=%d", r.MessageID), func() (any, error) {
			return bot.Messages.Delete(ctx, kappelas.DeleteMessageParams{
				ChatID:    chatID,
				MessageID: r.MessageID,
			})
		})
	}

	// ═══════════════════════════════════════════════════════════════════════════
	section("9. DELETE_PREVIOUS")
	// ═══════════════════════════════════════════════════════════════════════════

	dp1 := run("messages.Send() — message 1 (sera remplacé)", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "⏳ Message temporaire 1",
		})
	})
	if dp1 != nil {
		run("messages.Send() — DeletePrevious=true efface le précédent", func() (any, error) {
			return bot.Messages.Send(ctx, kappelas.SendMessageParams{
				ChatID:         chatID,
				Text:           "✅ Remplace le message précédent (DeletePrevious)",
				DeletePrevious: true,
			})
		})
	}

	// ═══════════════════════════════════════════════════════════════════════════
	section("10. WEBHOOKS")
	// ═══════════════════════════════════════════════════════════════════════════

	run("webhooks.GetInfo()", func() (any, error) {
		return bot.Webhooks.GetInfo(ctx)
	})

	skipTest("webhooks.Set()", "nécessite un serveur HTTPS public — skippé en local")
	skipTest("webhooks.Delete()", "skippé (aucun webhook actif à supprimer)")

	// ─── HandleWebhook — unit test sans serveur HTTP ──────────────────────────

	run("bot.HandleWebhook() — payload message", func() (any, error) {
		captured := make(chan *kappelas.Message, 1)
		const testMsgID = int64(99001)

		bot.OnMessage(func(msg *kappelas.Message) {
			if msg.ID == testMsgID {
				select {
				case captured <- msg:
				default:
				}
			}
		})

		payload, _ := json.Marshal(map[string]any{
			"type":       "text",
			"chat_id":    chatID,
			"message_id": testMsgID,
			"sender_id":  "test-user-uuid",
			"text":       "webhook test",
			"sent_at":    time.Now().Unix(),
		})
		bot.HandleWebhook(payload)

		select {
		case msg := <-captured:
			if msg.Text == nil || *msg.Text != "webhook test" {
				return nil, fmt.Errorf("texte incorrect : %v", msg.Text)
			}
			return map[string]any{"events_received": 1, "text": *msg.Text}, nil
		case <-time.After(300 * time.Millisecond):
			return nil, fmt.Errorf("timeout — aucun event reçu")
		}
	})

	run("bot.HandleWebhook() — payload callback_query", func() (any, error) {
		captured := make(chan *kappelas.CallbackQuery, 1)

		bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) {
			if cb.CallbackData == "webhook_cb_test_99002" {
				select {
				case captured <- cb:
				default:
				}
			}
		})

		payload, _ := json.Marshal(map[string]any{
			"type":             "callback",
			"chat_id":          chatID,
			"sender_id":        "test-user-uuid",
			"sender_nom":       "Test User",
			"sender_username":  "testuser",
			"callback_data":    "webhook_cb_test_99002",
			"sent_at":          time.Now().Unix(),
		})
		bot.HandleWebhook(payload)

		select {
		case cb := <-captured:
			if cb.CallbackData != "webhook_cb_test_99002" {
				return nil, fmt.Errorf("callback_data incorrect : %s", cb.CallbackData)
			}
			return map[string]any{"events_received": 1, "callback_data": cb.CallbackData}, nil
		case <-time.After(300 * time.Millisecond):
			return nil, fmt.Errorf("timeout — aucun event reçu")
		}
	})

	// ═══════════════════════════════════════════════════════════════════════════
	section("11. MEMBRES DU GROUPE (lecture seule)")
	// ═══════════════════════════════════════════════════════════════════════════

	if anyGroupID == 0 {
		skipTest("chats.GetAdministrators()", "aucun groupe trouvé")
		skipTest("chats.GetMember()", "aucun groupe trouvé")
		skipTest("chats.GetMember() — NOT_FOUND", "aucun groupe trouvé")
	} else {
		gid := anyGroupID

		var adminsResult *kappelas.GetChatAdministratorsResult
		if r, ok := run(fmt.Sprintf("chats.GetAdministrators({ ChatID: %d })", gid), func() (any, error) {
			return bot.Chats.GetAdministrators(ctx, kappelas.GetChatAdministratorsParams{ChatID: gid})
		}).(*kappelas.GetChatAdministratorsResult); ok && r != nil {
			adminsResult = r
			fmt.Printf("  [ℹ] %d admin(s)\n", len(r.Admins))
		}

		if adminsResult != nil && len(adminsResult.Admins) > 0 {
			first := adminsResult.Admins[0]
			run(fmt.Sprintf("chats.GetMember() — user_id=%s…", first.UserID[:8]), func() (any, error) {
				return bot.Chats.GetMember(ctx, kappelas.GetChatMemberParams{
					ChatID: gid,
					UserID: first.UserID,
				})
			})
		} else {
			skipTest("chats.GetMember()", "aucun admin trouvé pour tester")
		}

		// GetMember sur un user_id inexistant → doit lever NOT_FOUND
		runExpectError(
			"chats.GetMember() — user inexistant → NOT_FOUND",
			kappelas.ErrCodeNotFound,
			func() (any, error) {
				return bot.Chats.GetMember(ctx, kappelas.GetChatMemberParams{
					ChatID: gid,
					UserID: "00000000-0000-0000-0000-000000000000",
				})
			},
		)
	}

	// ═══════════════════════════════════════════════════════════════════════════
	section("12. INVITE LINKS (admin requis)")
	// ═══════════════════════════════════════════════════════════════════════════

	if adminGroupID == 0 {
		skipTest("chats.CreateInviteLink()", "aucun groupe admin")
		skipTest("chats.CreateSingleUseInviteLink()", "aucun groupe admin")
		skipTest("chats.CreateInviteLink() — max_uses+expires_in", "aucun groupe admin")
		skipTest("chats.GetInviteLinks()", "aucun groupe admin")
		skipTest("chats.RevokeInviteLink()", "aucun groupe admin")
	} else {
		gid := adminGroupID

		var permLink *kappelas.ChatInviteLink
		if r, ok := run("chats.CreateInviteLink() — permanent illimité", func() (any, error) {
			return bot.Chats.CreateInviteLink(ctx, kappelas.CreateChatInviteLinkParams{ChatID: gid})
		}).(*kappelas.ChatInviteLink); ok && r != nil {
			permLink = r
			fmt.Printf("  [ℹ] URL : %s\n", r.URL)
		}

		var singleLink *kappelas.ChatInviteLink
		if r, ok := run("chats.CreateSingleUseInviteLink() — usage unique", func() (any, error) {
			return bot.Chats.CreateSingleUseInviteLink(ctx, kappelas.CreateChatInviteLinkParams{ChatID: gid})
		}).(*kappelas.ChatInviteLink); ok && r != nil {
			singleLink = r
		}

		run("chats.CreateInviteLink() — max_uses=5, expires_in=24h", func() (any, error) {
			return bot.Chats.CreateInviteLink(ctx, kappelas.CreateChatInviteLinkParams{
				ChatID:    gid,
				MaxUses:   5,
				ExpiresIn: "24h",
			})
		})

		run("chats.GetInviteLinks() — liste les liens actifs", func() (any, error) {
			return bot.Chats.GetInviteLinks(ctx, kappelas.GetChatInviteLinksParams{ChatID: gid})
		})

		if permLink != nil {
			run(fmt.Sprintf("chats.RevokeInviteLink() — permanent code=%s", permLink.Code), func() (any, error) {
				return bot.Chats.RevokeInviteLink(ctx, kappelas.RevokeChatInviteLinkParams{
					ChatID: gid, Code: permLink.Code,
				})
			})
		}
		if singleLink != nil {
			run(fmt.Sprintf("chats.RevokeInviteLink() — single-use code=%s", singleLink.Code), func() (any, error) {
				return bot.Chats.RevokeInviteLink(ctx, kappelas.RevokeChatInviteLinkParams{
					ChatID: gid, Code: singleLink.Code,
				})
			})
		}
	}

	// ═══════════════════════════════════════════════════════════════════════════
	section("13. MESSAGES DANS LE GROUPE")
	// ═══════════════════════════════════════════════════════════════════════════

	if anyGroupID == 0 {
		skipTest("envoi dans le groupe", "aucun groupe trouvé")
	} else {
		gid := anyGroupID
		fmt.Printf("  [ℹ] Groupe cible : chat_id=%d\n", gid)

		groupMsg := run(fmt.Sprintf("messages.Send() — texte dans groupe %d", gid), func() (any, error) {
			return bot.Messages.Send(ctx, kappelas.SendMessageParams{
				ChatID: gid,
				Text:   "👋 Test SDK Go — message dans le groupe",
			})
		})

		if r, ok := groupMsg.(*kappelas.SendResult); ok && r != nil {
			run("messages.Send() — reply_to_id dans le groupe", func() (any, error) {
				return bot.Messages.Send(ctx, kappelas.SendMessageParams{
					ChatID:    gid,
					Text:      "↩️ Réponse avec citation dans le groupe",
					ReplyToID: &r.MessageID,
				})
			})

			run("messages.SendPhoto() — groupe avec reply_to_id", func() (any, error) {
				return bot.Messages.SendPhoto(ctx, kappelas.SendMediaParams{
					ChatID:    gid,
					File:      kappelas.FileInput{Data: pngBytes, Filename: "test.png", ContentType: "image/png"},
					Caption:   "🖼 Photo dans le groupe",
					ReplyToID: &r.MessageID,
				})
			})

			run("messages.SendCarousel() — groupe sans reply_to_id", func() (any, error) {
				return bot.Messages.SendCarousel(ctx, kappelas.SendCarouselParams{
					ChatID: gid,
					Text:   "🛍 Nos produits :",
					Carousel: []kappelas.CarouselCard{
						{ID: "g1", Title: "Produit A", Subtitle: ptr("9 990 FCFA"), ButtonText: ptr("Voir")},
						{ID: "g2", Title: "Produit B", Subtitle: ptr("19 990 FCFA"), ButtonText: ptr("Voir")},
					},
					QuickReplyButtons: []kappelas.ScrollKeyboardButton{
						{Text: "Voir plus"}, {Text: "Annuler"},
					},
				})
			})

			run("messages.SendCarousel() — groupe avec reply_to_id", func() (any, error) {
				return bot.Messages.SendCarousel(ctx, kappelas.SendCarouselParams{
					ChatID: gid,
					Text:   "↩️ Voici notre sélection en réponse :",
					Carousel: []kappelas.CarouselCard{
						{ID: "g3", Title: "Offre spéciale", Subtitle: ptr("4 990 FCFA"), ButtonText: ptr("Commander")},
					},
					QuickReplyButtons: []kappelas.ScrollKeyboardButton{
						{Text: "✅ Confirmer", CallbackData: "confirm"},
						{Text: "❌ Annuler", CallbackData: "cancel"},
					},
					ReplyToID: &r.MessageID,
				})
			})

			run("messages.Send() — inline keyboard dans le groupe", func() (any, error) {
				return bot.Messages.Send(ctx, kappelas.SendMessageParams{
					ChatID: gid,
					Text:   "Boutons inline dans le groupe :",
					ReplyMarkup: kappelas.InlineKeyboard{
						InlineKeyboard: [][]kappelas.InlineKeyboardButton{{
							{Text: "✅ Oui", CallbackData: ptr("group_yes")},
							{Text: "❌ Non", CallbackData: ptr("group_no")},
						}},
					},
				})
			})
		}
	}

	// ═══════════════════════════════════════════════════════════════════════════
	section("14. GESTION ERREURS")
	// ═══════════════════════════════════════════════════════════════════════════

	runExpectError(
		"messages.Send() vers chat_id invalide → FORBIDDEN",
		kappelas.ErrCodeForbidden,
		func() (any, error) {
			return bot.Messages.Send(ctx, kappelas.SendMessageParams{
				ChatID: -999999, Text: "test erreur",
			})
		},
	)

	// Delete d'un message inexistant — ne doit pas planter (succès silencieux)
	run("messages.Delete() message inexistant — pas d'erreur levée", func() (any, error) {
		return bot.Messages.Delete(ctx, kappelas.DeleteMessageParams{
			ChatID: chatID, MessageID: 999999999,
		})
	})

	runExpectError(
		"chats.CreateInviteLink() sur chat privé → FORBIDDEN",
		kappelas.ErrCodeForbidden,
		func() (any, error) {
			return bot.Chats.CreateInviteLink(ctx, kappelas.CreateChatInviteLinkParams{
				ChatID: chatID, // chat privé — bot n'est pas admin
			})
		},
	)

	// Vérifier les champs de KappelaError
	fmt.Printf("\n→ KappelaError — vérification des champs (Code, Status, Message, RequestID, Error())\n")
	_, errTest := bot.Messages.Send(ctx, kappelas.SendMessageParams{ChatID: -999999, Text: "err fields"})
	if errTest == nil {
		fmt.Printf("  [✗] FAIL — aurait dû retourner une erreur\n")
		failed++
	} else {
		var ke *kappelas.KappelaError
		if !errors.As(errTest, &ke) {
			fmt.Printf("  [✗] FAIL — pas une KappelaError : %v\n", errTest)
			failed++
		} else {
			ok := ke.Code != "" && ke.Status != 0 && ke.Message != "" && ke.Error() != ""
			if ok {
				fmt.Printf("  [✓] OK  KappelaError{Code:%s Status:%d hasMessage:%v hasError:%v}\n",
					ke.Code, ke.Status, ke.Message != "", ke.Error() != "")
				passed++
			} else {
				fmt.Printf("  [✗] FAIL — champs manquants : code=%q status=%d msg=%q\n",
					ke.Code, ke.Status, ke.Message)
				failed++
			}
		}
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// Résumé
	// ═══════════════════════════════════════════════════════════════════════════

	bot.Stop()

	total := passed + failed + skipped
	sep := strings.Repeat("═", 60)
	fmt.Printf("\n%s\n  Résultats : %d passés  %d échoués  %d ignorés  (%d total)\n%s\n",
		sep, passed, failed, skipped, total, sep)

	if failed > 0 {
		os.Exit(1)
	}
}
