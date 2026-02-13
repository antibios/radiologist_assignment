package main

import (
	"context"
	"fmt"
	"os"
	"net/http"
	"net/http/httptest"
	"radiology-assignment/internal/assignment"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestE2E(t *testing.T) {
	// Initialize engine for testing as main() is skipped
	engine = assignment.NewEngine(&InMemoryStore{}, &InMemoryRoster{}, &InMemoryRules{})

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
		case "/api/simulate":
			handleSimulateAssignment(w, r)
		case "/active_search":
			handleActiveSearch(w, r)
		case "/calendar":
			handleCalendar(w, r)
		default:
			if strings.HasPrefix(r.URL.Path, "/static/") {
				dir := "ui/static"
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					dir = "../../ui/static"
				}
				http.StripPrefix("/static/", http.FileServer(http.Dir(dir))).ServeHTTP(w, r)
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

	t.Run("CreateRule", func(t *testing.T) {
		ruleName := "E2E Test Rule"
		var res string

		err := chromedp.Run(ctx,
			chromedp.Navigate(ts.URL+"/rules"),
			// Check for 500 error on page load by looking for common error text or ensuring expected element exists
			chromedp.WaitVisible(`button[onclick="ui('#add-rule-modal')"]`, chromedp.ByQuery),
			// Force modal open
			chromedp.Evaluate(`document.getElementById('add-rule-modal').setAttribute('open', 'true')`, nil),
			chromedp.Sleep(500*time.Millisecond),
			chromedp.WaitVisible(`#add-rule-modal input[name="name"]`, chromedp.ByQuery),
			chromedp.SendKeys(`#add-rule-modal input[name="name"]`, ruleName, chromedp.ByQuery),
			chromedp.Click(`#add-rule-modal button[type="submit"]`, chromedp.ByQuery),
			chromedp.WaitVisible(`.rule-name`, chromedp.ByQuery),
			chromedp.Text(`//td[contains(@class, "rule-name") and text()="`+ruleName+`"]`, &res),
		)

		if err != nil {
			t.Fatalf("Failed to create rule: %v", err)
		}
		if res != ruleName {
			t.Errorf("Expected rule name %s, got %s", ruleName, res)
		}
	})

	t.Run("CreateShift", func(t *testing.T) {
		shiftName := "E2E Test Shift"
		var res string

		err := chromedp.Run(ctx,
			chromedp.Navigate(ts.URL+"/shifts"),
			// Wait for button
			chromedp.WaitVisible(`button[onclick="ui('#add-shift-modal')"]`, chromedp.ByQuery),
			// Click to open modal
			chromedp.Click(`button[onclick="ui('#add-shift-modal')"]`, chromedp.ByQuery),
			chromedp.Sleep(500*time.Millisecond),
			// Wait for input
			chromedp.WaitVisible(`#add-shift-modal input[name="name"]`, chromedp.ByQuery),
			// Fill Form
			chromedp.SendKeys(`#add-shift-modal input[name="name"]`, shiftName, chromedp.ByQuery),
			chromedp.SendKeys(`#add-shift-modal select[name="work_type"]`, "CT", chromedp.ByQuery),
			// Use active search for Site (Select SiteA)
			chromedp.SetValue(`#site-search-container input`, "SiteA", chromedp.ByQuery),
			chromedp.Evaluate(`document.querySelector('#site-search-container input').dispatchEvent(new Event('input'))`, nil),
			chromedp.Sleep(2*time.Second), // Wait for debounce and fetch
			chromedp.WaitVisible(`#site-results a`, chromedp.ByQuery),
			chromedp.Click(`#site-results a`, chromedp.ByQuery),

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

		err := chromedp.Run(ctx,
			chromedp.Navigate(ts.URL+"/shifts"),
			chromedp.Click(fmt.Sprintf(`//tr[td[contains(text(), "%s")]]//button[contains(@onclick, "openAssignModal")]`, shiftName), chromedp.BySearch),
			chromedp.Sleep(500*time.Millisecond),
			// Search for radiologist rad1
			chromedp.WaitVisible(`#radiologist-search-container input`, chromedp.ByQuery),
			chromedp.SetValue(`#radiologist-search-container input`, "rad1", chromedp.ByQuery),
			chromedp.Evaluate(`document.querySelector('#radiologist-search-container input').dispatchEvent(new Event('input'))`, nil),
			chromedp.Sleep(2*time.Second),
			chromedp.WaitVisible(`#radiologist-results a`, chromedp.ByQuery),
			chromedp.Click(`#radiologist-results a`, chromedp.ByQuery),

			chromedp.Click(`#assign-rad-modal button[type="submit"]`, chromedp.ByQuery),
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
			chromedp.Sleep(500*time.Millisecond),
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

	t.Run("TestCalendarView", func(t *testing.T) {
		err := chromedp.Run(ctx,
			chromedp.Navigate(ts.URL+"/calendar"),
			// Check for Month View Default
			chromedp.WaitVisible(`.grid-calendar`, chromedp.ByQuery),
			// Navigate to Week View
			chromedp.Click(`a[href="?view=week"]`, chromedp.ByQuery),
			// Check for Table
			chromedp.WaitVisible(`table.stripes`, chromedp.ByQuery),
		)
		if err != nil {
			t.Fatalf("Failed calendar view test: %v", err)
		}
	})
}
