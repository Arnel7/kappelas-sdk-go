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

	// Répondre aux clics de boutons pendant les tests
	bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) {
		fmt.Printf("\n[→] Bouton cliqué — chat_id=%d sender=%q data=%q\n",
			cb.ChatID, ptrStr(cb.SenderNom, cb.SenderID), cb.CallbackData)
		if _, err := bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: cb.ChatID,
			Text:   "Tu as cliqué : " + cb.CallbackData,
		}); err != nil {
			fmt.Printf("[✗] Erreur réponse callback : %v\n", err)
		}
	})

	bot.Start()

	select {
	case <-connected:
		fmt.Printf("[✓] Connecté — chat_id cible : %d\n\n", chatID)
	case <-time.After(10 * time.Second):
		fmt.Println("[✗] Timeout connexion WebSocket")
		os.Exit(1)
	}

	// ─── Tests ───────────────────────────────────────────────────────────────

	// 1. Profil
	run("profile.Get()", func() (any, error) {
		return bot.Profile.Get(ctx)
	})

	// 2. Chats
	run("chats.List(limit=3)", func() (any, error) {
		return bot.Chats.List(ctx, kappelas.GetChatsParams{Limit: 3})
	})

	// 3. Texte simple
	sent := run("messages.Send() — texte simple", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "👋 Test SDK Go — message texte",
		})
	})

	// 4. Inline keyboard
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

	// 5. Reply keyboard
	run("messages.Send() — reply keyboard", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Test reply keyboard :",
			ReplyMarkup: kappelas.ReplyKeyboard{
				Keyboard: [][]string{{"Option A", "Option B"}, {"Annuler"}},
			},
		})
	})

	// 6. Scroll keyboard
	run("messages.Send() — scroll keyboard", func() (any, error) {
		return bot.Messages.Send(ctx, kappelas.SendMessageParams{
			ChatID: chatID,
			Text:   "Test scroll keyboard :",
			ReplyMarkup: kappelas.ScrollKeyboard{
				ScrollKeyboard: []string{"Petit", "Moyen", "Grand", "XL"},
			},
		})
	})

	// 7. Typing indicator
	run("messages.SendTyping() — show", func() (any, error) {
		return bot.Messages.SendTyping(ctx, kappelas.SendTypingParams{ChatID: chatID})
	})
	hide := false
	run("messages.SendTyping() — hide", func() (any, error) {
		return bot.Messages.SendTyping(ctx, kappelas.SendTypingParams{ChatID: chatID, IsTyping: &hide})
	})

	// 8. Photo
	run("messages.SendPhoto()", func() (any, error) {
		return bot.Messages.SendPhoto(ctx, kappelas.SendMediaParams{
			ChatID:  chatID,
			File:    kappelas.FileInput{Data: pngBytes, Filename: "test.png", ContentType: "image/png"},
			Caption: "Test photo depuis le SDK Go",
		})
	})

	// 9. Document
	run("messages.SendDocument()", func() (any, error) {
		return bot.Messages.SendDocument(ctx, kappelas.SendMediaParams{
			ChatID:  chatID,
			File:    kappelas.FileInput{Data: pdfBytes, Filename: "test.pdf", ContentType: "application/pdf"},
			Caption: "Test document depuis le SDK Go",
		})
	})

	// 10. Audio
	run("messages.SendAudio()", func() (any, error) {
		return bot.Messages.SendAudio(ctx, kappelas.SendMediaParams{
			ChatID:  chatID,
			File:    kappelas.FileInput{Data: wavBytes, Filename: "test.wav", ContentType: "audio/wav"},
			Caption: "Test audio depuis le SDK Go",
		})
	})

	// 11. Carousel
	run("messages.SendCarousel()", func() (any, error) {
		return bot.Messages.SendCarousel(ctx, kappelas.SendCarouselParams{
			ChatID: chatID,
			Text:   "Test carousel :",
			Carousel: []kappelas.CarouselCard{
				{ID: "p1", Title: "Produit A", Subtitle: ptr("9,99 €"), ButtonText: ptr("Voir")},
				{ID: "p2", Title: "Produit B", Subtitle: ptr("19,99 €"), ButtonText: ptr("Voir")},
			},
			QuickReplyButtons: []string{"Voir plus", "Annuler"},
		})
	})

	// 12. Send + Edit
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

	// 13. Delete
	if r, ok := sent.(*kappelas.SendResult); ok && r != nil {
		run(fmt.Sprintf("messages.Delete() — message_id=%d", r.MessageID), func() (any, error) {
			return bot.Messages.Delete(ctx, kappelas.DeleteMessageParams{
				ChatID:    chatID,
				MessageID: r.MessageID,
			})
		})
	}

	// 14. Webhook info
	run("webhooks.GetInfo()", func() (any, error) {
		return bot.Webhooks.GetInfo(ctx)
	})

	bot.Stop()

	fmt.Printf("\n%s\n", "────────────────────────────────────────")
	fmt.Printf("[✓] %d passés   [✗] %d échoués\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
