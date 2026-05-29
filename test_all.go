//go:build ignore

// Tests live du SDK Kappelas Go.
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
	passed int
	failed int
	ctx    = context.Background()
)

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
	fmt.Printf("  [✓] OK  %s\n", b)
	passed++
	return result
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
		fmt.Printf("[!] Erreur : %v\n", err)
	})

	// Capturer les messages entrants pendant les tests (pour tester bot.Reply)
	var lastMsg *kappelas.Message
	bot.OnMessage(func(msg *kappelas.Message) {
		lastMsg = msg
		fmt.Printf("\n[→] Message reçu — chat_id=%d type=%s\n", msg.ChatID, msg.Type)
	})

	// Répondre aux clics de boutons pendant les tests
	bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) {
		fmt.Printf("\n[→] Bouton cliqué — chat_id=%d sender=%q data=%q\n",
			cb.ChatID, ptrStr(cb.SenderNom, cb.SenderID), cb.CallbackData)
		bot.Reply(ctx, cb, "Tu as cliqué : "+cb.CallbackData)
	})

	bot.Start()

	select {
	case <-connected:
		fmt.Printf("[✓] Connecté — chat_id cible : %d\n\n", chatID)
	case <-time.After(10 * time.Second):
		fmt.Println("[✗] Timeout connexion WebSocket")
		os.Exit(1)
	}

	// ─── 1. Profil ───────────────────────────────────────────────────────────

	run("profile.Get()", func() (any, error) {
		return bot.Profile.Get(ctx)
	})

	// ─── 2. Chats ────────────────────────────────────────────────────────────

	run("chats.List(limit=3)", func() (any, error) {
		return bot.Chats.List(ctx, kappelas.GetChatsParams{Limit: 3})
	})

	// ─── 3. Texte simple ─────────────────────────────────────────────────────

	sent := run("messages.Send() — texte simple", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "👋 Test SDK Go — message texte",
		})
	})

	// ─── 4. Inline keyboard ──────────────────────────────────────────────────

	run("messages.Send() — inline keyboard", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Test avec boutons inline :",
			ReplyMarkup: kappelas.InlineKeyboard{
				InlineKeyboard: [][]kappelas.InlineKeyboardButton{{
					{Text: "✅ Oui", CallbackData: ptr("yes")},
					{Text: "❌ Non", CallbackData: ptr("no")},
				}},
			},
		})
	})

	// ─── 5. Reply keyboard (short form) ──────────────────────────────────────

	run("messages.Send() — reply keyboard (short form)", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Test reply keyboard (short form) :",
			ReplyMarkup: kappelas.ReplyKeyboard{
				Keyboard: [][]kappelas.ReplyKeyboardButton{
					{{Text: "Option A"}, {Text: "Option B"}},
					{{Text: "Annuler"}},
				},
			},
		})
	})

	// ─── 6. Reply keyboard (long form) ───────────────────────────────────────

	run("messages.Send() — reply keyboard (long form)", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Test reply keyboard (long form — label ≠ callback) :",
			ReplyMarkup: kappelas.ReplyKeyboard{
				Keyboard: [][]kappelas.ReplyKeyboardButton{
					{
						{Text: "✅ Confirmer", CallbackData: "confirm_yes"},
						{Text: "❌ Annuler", CallbackData: "confirm_no"},
					},
				},
			},
		})
	})

	// ─── 7. Scroll keyboard (short form) ─────────────────────────────────────

	run("messages.Send() — scroll keyboard (short form)", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Test scroll keyboard :",
			ReplyMarkup: kappelas.ScrollKeyboard{
				ScrollKeyboard: []kappelas.ScrollKeyboardButton{
					{Text: "Petit"}, {Text: "Moyen"}, {Text: "Grand"}, {Text: "XL"},
				},
			},
		})
	})

	// ─── 8. Scroll keyboard (long form) ──────────────────────────────────────

	run("messages.Send() — scroll keyboard (long form)", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Test scroll keyboard (long form) :",
			ReplyMarkup: kappelas.ScrollKeyboard{
				ScrollKeyboard: []kappelas.ScrollKeyboardButton{
					{Text: "📦 Commandes", CallbackData: "menu_orders"},
					{Text: "❓ Aide", CallbackData: "menu_help"},
				},
			},
		})
	})

	// ─── 9. Typing indicator ─────────────────────────────────────────────────

	run("messages.SendTyping() — show", func() (any, error) {
		return bot.Messages.SendTyping(ctx, kappelas.SendTypingParams{ChatID: chatID})
	})
	hide := false
	run("messages.SendTyping() — hide", func() (any, error) {
		return bot.Messages.SendTyping(ctx, kappelas.SendTypingParams{ChatID: chatID, IsTyping: &hide})
	})

	// ─── 10. Photo ───────────────────────────────────────────────────────────

	run("messages.SendPhoto()", func() (any, error) {
		return bot.Messages.SendPhoto(ctx, kappelas.SendMediaParams{
			ChatID:  chatID,
			File:    kappelas.FileInput{Data: pngBytes, Filename: "test.png", ContentType: "image/png"},
			Caption: "Test photo depuis le SDK Go",
		})
	})

	// ─── 11. Document ────────────────────────────────────────────────────────

	run("messages.SendDocument()", func() (any, error) {
		return bot.Messages.SendDocument(ctx, kappelas.SendMediaParams{
			ChatID:  chatID,
			File:    kappelas.FileInput{Data: pdfBytes, Filename: "test.pdf", ContentType: "application/pdf"},
			Caption: "Test document depuis le SDK Go",
		})
	})

	// ─── 12. Audio ───────────────────────────────────────────────────────────

	run("messages.SendAudio()", func() (any, error) {
		return bot.Messages.SendAudio(ctx, kappelas.SendMediaParams{
			ChatID:  chatID,
			File:    kappelas.FileInput{Data: wavBytes, Filename: "test.wav", ContentType: "audio/wav"},
			Caption: "Test audio depuis le SDK Go",
		})
	})

	// ─── 13. Carousel ────────────────────────────────────────────────────────

	run("messages.SendCarousel()", func() (any, error) {
		return bot.Messages.SendCarousel(ctx, kappelas.SendCarouselParams{
			ChatID: chatID,
			Text:   "Test carousel :",
			Carousel: []kappelas.CarouselCard{
				{ID: "p1", Title: "Produit A", Subtitle: ptr("9 900 FCFA"), ButtonText: ptr("Voir")},
				{ID: "p2", Title: "Produit B", Subtitle: ptr("19 900 FCFA"), ButtonText: ptr("Voir")},
			},
			QuickReplyButtons: []kappelas.ScrollKeyboardButton{
				{Text: "Voir plus"}, {Text: "Annuler"},
			},
		})
	})

	// ─── 14. Edit ────────────────────────────────────────────────────────────

	sentEdit := run("messages.Send() — pour edit", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Message à modifier :",
			ReplyMarkup: kappelas.InlineKeyboard{
				InlineKeyboard: [][]kappelas.InlineKeyboardButton{{
					{Text: "🔴 Avant", CallbackData: ptr("before")},
				}},
			},
		})
	})
	if r, ok := sentEdit.(*kappelas.SendResult); ok && r != nil {
		run("messages.Edit() — texte seul", func() (any, error) {
			return bot.Messages.Edit(ctx, kappelas.EditMessageParams{
				ChatID:    chatID,
				MessageID: r.MessageID,
				NewText:   "Message modifié ✅",
			})
		})
	}

	// ─── 15. Delete ──────────────────────────────────────────────────────────

	if r, ok := sent.(*kappelas.SendResult); ok && r != nil {
		run(fmt.Sprintf("messages.Delete() — message_id=%d", r.MessageID), func() (any, error) {
			return bot.Messages.Delete(ctx, kappelas.DeleteMessageParams{
				ChatID:    chatID,
				MessageID: r.MessageID,
			})
		})
	}

	// ─── 16. bot.Reply() ─────────────────────────────────────────────────────

	// Envoyer un message, attendre un peu pour le recevoir par WS, puis Reply
	var replySource *kappelas.SendResult
	if r, ok := run("messages.Send() — source pour bot.Reply()", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Message source pour tester bot.Reply() 📌",
		})
	}).(*kappelas.SendResult); ok && r != nil {
		replySource = r
	}

	// Attendre que le WS livre le message (max 3s)
	if replySource != nil {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) && lastMsg == nil {
			time.Sleep(100 * time.Millisecond)
		}
	}

	if lastMsg != nil {
		run("bot.Reply() — depuis *Message", func() (any, error) {
			return bot.Reply(ctx, lastMsg, "↩️ Réponse via bot.Reply()")
		})
		run("bot.Reply() — avec inline keyboard", func() (any, error) {
			return bot.Reply(ctx, lastMsg, "Choix via bot.Reply() :", kappelas.SendMessageParams{
				ReplyMarkup: kappelas.InlineKeyboard{
					InlineKeyboard: [][]kappelas.InlineKeyboardButton{{
						{Text: "✅ OK", CallbackData: ptr("reply_ok")},
					}},
				},
			})
		})
	} else {
		fmt.Println("\n→ bot.Reply() — skipped (no WS message received in time)")
	}

	// ─── 17. Webhook info ────────────────────────────────────────────────────

	run("webhooks.GetInfo()", func() (any, error) {
		return bot.Webhooks.GetInfo(ctx)
	})

	// ─── 18. chats.GetMyGroups() ─────────────────────────────────────────────

	var myGroups *kappelas.GetMyGroupsResult
	if r, ok := run("chats.GetMyGroups()", func() (any, error) {
		return bot.Chats.GetMyGroups(ctx)
	}).(*kappelas.GetMyGroupsResult); ok && r != nil {
		myGroups = r
		fmt.Printf("  [i] %d groupe(s)/canal/canaux\n", len(myGroups.Groups))
		for _, g := range myGroups.Groups {
			title := "(sans titre)"
			if g.Title != nil {
				title = *g.Title
			}
			fmt.Printf("      %d (%s) %q → %s\n", g.ChatID, g.Type, title, g.BotRole)
		}
	}

	// ─── 19. Admin-only group tests (si le bot est admin dans un groupe) ──────

	// Utilise le premier groupe où le bot est admin, sinon saute les tests
	var adminGroupID int64
	if myGroups != nil {
		for _, g := range myGroups.Groups {
			if g.BotRole == kappelas.ParticipantRoleAdmin {
				adminGroupID = g.ChatID
				break
			}
		}
	}

	if adminGroupID != 0 {
		fmt.Printf("\n[i] Tests admin dans le groupe %d\n", adminGroupID)

		// GetAdministrators
		var admins *kappelas.GetChatAdministratorsResult
		if r, ok := run("chats.GetAdministrators()", func() (any, error) {
			return bot.Chats.GetAdministrators(ctx, kappelas.GetChatAdministratorsParams{
				ChatID: adminGroupID,
			})
		}).(*kappelas.GetChatAdministratorsResult); ok && r != nil {
			admins = r
			fmt.Printf("  [i] %d admin(s)\n", len(admins.Admins))
		}

		// GetMember — cherche le premier admin
		if admins != nil && len(admins.Admins) > 0 {
			firstAdmin := admins.Admins[0]
			run(fmt.Sprintf("chats.GetMember() — user_id=%s", firstAdmin.UserID), func() (any, error) {
				return bot.Chats.GetMember(ctx, kappelas.GetChatMemberParams{
					ChatID: adminGroupID,
					UserID: firstAdmin.UserID,
				})
			})
		}

		// CreateInviteLink — permanent, unlimited
		var inviteLink *kappelas.ChatInviteLink
		if r, ok := run("chats.CreateInviteLink() — permanent", func() (any, error) {
			return bot.Chats.CreateInviteLink(ctx, kappelas.CreateChatInviteLinkParams{
				ChatID: adminGroupID,
			})
		}).(*kappelas.ChatInviteLink); ok && r != nil {
			inviteLink = r
			fmt.Printf("  [i] URL : %s\n", inviteLink.URL)
		}

		// GetInviteLinks
		run("chats.GetInviteLinks()", func() (any, error) {
			return bot.Chats.GetInviteLinks(ctx, kappelas.GetChatInviteLinksParams{
				ChatID: adminGroupID,
			})
		})

		// CreateSingleUseInviteLink
		var singleLink *kappelas.ChatInviteLink
		if r, ok := run("chats.CreateSingleUseInviteLink()", func() (any, error) {
			return bot.Chats.CreateSingleUseInviteLink(ctx, kappelas.CreateChatInviteLinkParams{
				ChatID: adminGroupID,
			})
		}).(*kappelas.ChatInviteLink); ok && r != nil {
			singleLink = r
		}

		// RevokeInviteLink — révoque le lien à usage unique
		if singleLink != nil {
			run(fmt.Sprintf("chats.RevokeInviteLink() — code=%s", singleLink.Code), func() (any, error) {
				return bot.Chats.RevokeInviteLink(ctx, kappelas.RevokeChatInviteLinkParams{
					ChatID: adminGroupID,
					Code:   singleLink.Code,
				})
			})
		}

		// Révoquer aussi le lien permanent créé
		if inviteLink != nil {
			run("chats.RevokeInviteLink() — permanent", func() (any, error) {
				return bot.Chats.RevokeInviteLink(ctx, kappelas.RevokeChatInviteLinkParams{
					ChatID: adminGroupID,
					Code:   inviteLink.Code,
				})
			})
		}
	} else {
		fmt.Println("\n[i] Tests admin skipped — le bot n'est admin dans aucun groupe")
		fmt.Println("    (ajouter le bot comme admin pour tester GetAdministrators, invite links, etc.)")
	}

	bot.Stop()

	fmt.Printf("\n%s\n", "────────────────────────────────────────")
	fmt.Printf("[✓] %d passés   [✗] %d échoués\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
