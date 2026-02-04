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
		case "/api/simulate":
			handleSimulateAssignment(w, r)
		case "/calendar":
			handleCalendar(w, r)
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

	t.Run("CreateRule", func(t *testing.T) {
		ruleName := "E2E Test Rule"
		var res string

		err := chromedp.Run(ctx,
			chromedp.Navigate(ts.URL+"/rules"),
			// Check for 500 error on page load by looking for common error text or ensuring expected element exists
			chromedp.WaitVisible(`button[onclick="ui('#add-rule-modal')"]`, chromedp.ByQuery),
			// Force modal open
			chromedp.Evaluate(`document.getElementById('add-rule-modal').setAttribute('open', 'true')`, nil),
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

		err := chromedp.Run(ctx,
			chromedp.Navigate(ts.URL+"/shifts"),
			chromedp.Click(fmt.Sprintf(`//tr[td[contains(text(), "%s")]]//button[contains(@onclick, "openAssignModal")]`, shiftName), chromedp.BySearch),
			chromedp.WaitVisible(`#assign-rad-modal select[name="radiologist_id"]`, chromedp.ByQuery),
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

	t.Run("TestOverAssignment", func(t *testing.T) {
		shiftName := "Overload Shift"

		err := chromedp.Run(ctx,
			chromedp.Navigate(ts.URL+"/shifts"),
			// Create Shift
			chromedp.Click(`button[onclick="ui('#add-shift-modal')"]`, chromedp.ByQuery),
			chromedp.WaitVisible(`#add-shift-modal input[name="name"]`, chromedp.ByQuery),
			chromedp.SendKeys(`#add-shift-modal input[name="name"]`, shiftName, chromedp.ByQuery),
			chromedp.SendKeys(`#add-shift-modal input[name="work_type"]`, "XRAY", chromedp.ByQuery),
			chromedp.SendKeys(`#add-shift-modal input[name="site"]`, "SiteA", chromedp.ByQuery),
			chromedp.Click(`#add-shift-modal button[type="submit"]`, chromedp.ByQuery),
			chromedp.WaitVisible(fmt.Sprintf(`//td[contains(@class, "shift-name") and text()="%s"]`, shiftName), chromedp.BySearch),

			// Assign rad_limited
			chromedp.Click(fmt.Sprintf(`//tr[td[contains(text(), "%s")]]//button[contains(@onclick, "openAssignModal")]`, shiftName), chromedp.BySearch),
			chromedp.WaitVisible(`#assign-rad-modal select[name="radiologist_id"]`, chromedp.ByQuery),
			chromedp.SetValue(`#assign-rad-modal select[name="radiologist_id"]`, "rad_limited", chromedp.ByQuery),
			chromedp.Click(`#assign-rad-modal button[type="submit"]`, chromedp.ByQuery),
			chromedp.WaitVisible(fmt.Sprintf(`//tr[td[contains(text(), "%s")]]//span[contains(text(), "rad_limited")]`, shiftName), chromedp.BySearch),
		)
		if err != nil {
			t.Fatalf("Failed setup for OverAssignment: %v", err)
		}

		// Simulate Assignment 1 via Fetch
		var result1 string
		err = chromedp.Run(ctx,
			chromedp.Evaluate(`fetch('/api/simulate', {
				method: 'POST',
				headers: {'Content-Type': 'application/x-www-form-urlencoded'},
				body: 'study_id=STUDY_1&modality=XRAY'
			}).then(r => r.text())`, &result1),
		)
		if err != nil {
			t.Fatalf("Failed simulation 1: %v", err)
		}
		if !strings.Contains(result1, "Assigned to rad_limited") {
			t.Errorf("Expected assignment to rad_limited, got: %s", result1)
		}

		// Simulate Assignment 2 via Fetch
		var resMap map[string]interface{}
		err = chromedp.Run(ctx,
			chromedp.Evaluate(`fetch('/api/simulate', {
				method: 'POST',
				headers: {'Content-Type': 'application/x-www-form-urlencoded'},
				body: 'study_id=STUDY_2&modality=XRAY'
			}).then(r => r.text().then(t => ({status: r.status, text: t})))`, &resMap),
		)

		if err != nil {
			t.Fatalf("Failed simulation 2: %v", err)
		}

		status := int(resMap["status"].(float64))
		text := resMap["text"].(string)

		if status != 503 {
			t.Errorf("Expected 503 Service Unavailable (Over Capacity), got %d: %s", status, text)
		}
	})
}
