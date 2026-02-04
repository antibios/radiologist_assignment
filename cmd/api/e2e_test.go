package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestE2E(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			handleDashboard(w, r)
		case "/rules":
			handleRules(w, r)
		case "/api/rules":
			handleAPIRules(w, r)
		case "/api/rules/edit":
			handleEditRule(w, r)
		case "/api/rules/delete":
			handleDeleteRule(w, r)
		case "/shifts":
			handleShifts(w, r)
		case "/api/shifts":
			handleAPIShifts(w, r)
		case "/api/shifts/edit":
			handleEditShift(w, r)
		case "/api/shifts/delete":
			handleDeleteShift(w, r)
		case "/api/shifts/assign":
			handleAssignRadiologist(w, r)
		default:
			if strings.HasPrefix(r.URL.Path, "/static/") {
				http.StripPrefix("/static/", http.FileServer(http.Dir("ui/static"))).ServeHTTP(w, r)
				return
			}
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	t.Run("CreateShift", func(t *testing.T) {
		shiftName := "E2E Test Shift"
		var res string

		err := chromedp.Run(ctx,
			chromedp.Navigate(ts.URL+"/shifts"),
			// Wait for button
			chromedp.WaitVisible(`button[onclick="ui('#add-shift-modal')"]`, chromedp.ByQuery),
			// Click to open modal
			chromedp.Click(`button[onclick="ui('#add-shift-modal')"]`, chromedp.ByQuery),
			// Wait for input
			chromedp.WaitVisible(`#add-shift-modal input[name="name"]`, chromedp.ByQuery),
			// Fill Form
			chromedp.SendKeys(`#add-shift-modal input[name="name"]`, shiftName, chromedp.ByQuery),
			chromedp.SendKeys(`#add-shift-modal input[name="work_type"]`, "CT", chromedp.ByQuery),
			chromedp.SendKeys(`#add-shift-modal input[name="site"]`, "Metro", chromedp.ByQuery),
			chromedp.SendKeys(`#add-shift-modal input[name="credentials"]`, "CT,Neuro", chromedp.ByQuery),
			// Submit
			chromedp.Click(`#add-shift-modal button[type="submit"]`, chromedp.ByQuery),
			// Verify
			chromedp.WaitVisible(`.shift-name`, chromedp.ByQuery),
			chromedp.Text(`//td[contains(@class, "shift-name") and text()="`+shiftName+`"]`, &res),
		)

		if err != nil {
			t.Fatalf("Failed to create shift: %v", err)
		}
		if res != shiftName {
			t.Errorf("Expected shift name %s, got %s", shiftName, res)
		}
	})

	t.Run("AssignRadiologist", func(t *testing.T) {
		shiftName := "E2E Test Shift"
		// We want to assign a radiologist to the shift created above.
		// We find the row, click "person_add" button.

		err := chromedp.Run(ctx,
			chromedp.Navigate(ts.URL+"/shifts"),
			// Click Assign button in the row of our shift
			chromedp.Click(fmt.Sprintf(`//tr[td[contains(text(), "%s")]]//button[contains(@onclick, "openAssignModal")]`, shiftName), chromedp.BySearch),
			// Wait for modal
			chromedp.WaitVisible(`#assign-rad-modal select[name="radiologist_id"]`, chromedp.ByQuery),
			// Select first option (should be rad1)
			// Actually we just submit defaults to "rad1"
			chromedp.Click(`#assign-rad-modal button[type="submit"]`, chromedp.ByQuery),
			// Verify assignment appears in table
			chromedp.WaitVisible(fmt.Sprintf(`//tr[td[contains(text(), "%s")]]//span[contains(text(), "rad1")]`, shiftName), chromedp.BySearch),
		)

		if err != nil {
			t.Fatalf("Failed to assign radiologist: %v", err)
		}
	})

	t.Run("EditShift", func(t *testing.T) {
		newShiftName := "E2E Test Shift Edited"

		err := chromedp.Run(ctx,
			chromedp.Navigate(ts.URL+"/shifts"),
			chromedp.Click(fmt.Sprintf(`//tr[td[contains(text(), "E2E Test Shift")]]//button[contains(@onclick, "openEditShiftModal")]`), chromedp.BySearch),
			chromedp.WaitVisible(`#edit-shift-modal input[name="name"]`, chromedp.ByQuery),
			chromedp.SetValue(`#edit-shift-name`, newShiftName, chromedp.ByQuery),
			chromedp.Click(`#edit-shift-modal button[type="submit"]`, chromedp.ByQuery),
			chromedp.WaitVisible(`//td[contains(text(), "`+newShiftName+`")]`, chromedp.BySearch),
		)

		if err != nil {
			t.Fatalf("Failed to edit shift: %v", err)
		}
	})

	t.Run("DeleteShift", func(t *testing.T) {
		shiftName := "E2E Test Shift Edited"

		err := chromedp.Run(ctx,
			chromedp.Navigate(ts.URL+"/shifts"),
			chromedp.WaitVisible(`//td[contains(text(), "`+shiftName+`")]`, chromedp.BySearch),
			chromedp.Click(fmt.Sprintf(`//tr[td[contains(text(), "%s")]]//form[contains(@action, "delete")]//button`, shiftName), chromedp.BySearch),
			chromedp.WaitNotPresent(`//td[contains(text(), "`+shiftName+`")]`, chromedp.BySearch),
		)

		if err != nil {
			t.Fatalf("Failed to delete shift: %v", err)
		}
	})
}
