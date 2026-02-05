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
			middleware.CSRF(handleDashboard)(w, r)
		case "/rules":
			middleware.CSRF(handleRules)(w, r)
		case "/api/rules":
			middleware.CSRF(handleAPIRules)(w, r)
		case "/api/rules/edit":
			middleware.CSRF(handleEditRule)(w, r)
		case "/api/rules/delete":
			middleware.CSRF(handleDeleteRule)(w, r)
		case "/shifts":
			middleware.CSRF(handleShifts)(w, r)
		case "/api/shifts":
			middleware.CSRF(handleAPIShifts)(w, r)
		case "/api/shifts/edit":
			middleware.CSRF(handleEditShift)(w, r)
		case "/api/shifts/delete":
			middleware.CSRF(handleDeleteShift)(w, r)
		case "/api/shifts/assign":
			middleware.CSRF(handleAssignRadiologist)(w, r)
		case "/radiologists":
			middleware.CSRF(handleRadiologists)(w, r)
		case "/api/radiologists":
			middleware.CSRF(handleAPIRadiologists)(w, r)
		case "/api/radiologists/edit":
			middleware.CSRF(handleEditRadiologist)(w, r)
		case "/api/radiologists/delete":
			middleware.CSRF(handleDeleteRadiologist)(w, r)
		case "/calendar":
			middleware.CSRF(handleCalendar)(w, r)
		case "/api/simulate":
			middleware.CSRF(handleSimulateAssignment)(w, r)
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
			chromedp.WaitVisible(`button[onclick="ui('#add-rule-modal')"]`, chromedp.ByQuery),
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

	t.Run("TestOverAssignment", func(t *testing.T) {
		shiftName := "Overload Shift"

		err := chromedp.Run(ctx,
			chromedp.Navigate(ts.URL+"/shifts"),
			chromedp.Click(`button[onclick="ui('#add-shift-modal')"]`, chromedp.ByQuery),
			chromedp.WaitVisible(`#add-shift-modal input[name="name"]`, chromedp.ByQuery),
			chromedp.SendKeys(`#add-shift-modal input[name="name"]`, shiftName, chromedp.ByQuery),
			chromedp.SendKeys(`#add-shift-modal input[name="work_type"]`, "XRAY", chromedp.ByQuery),
			chromedp.SendKeys(`#add-shift-modal input[name="site"]`, "SiteA", chromedp.ByQuery),
			chromedp.Click(`#add-shift-modal button[type="submit"]`, chromedp.ByQuery),
			chromedp.WaitVisible(fmt.Sprintf(`//td[contains(@class, "shift-name") and text()="%s"]`, shiftName), chromedp.BySearch),

			chromedp.Click(fmt.Sprintf(`//tr[td[contains(text(), "%s")]]//button[contains(@onclick, "openAssignModal")]`, shiftName), chromedp.BySearch),
			chromedp.WaitVisible(`#assign-rad-modal select[name="radiologist_id"]`, chromedp.ByQuery),
			chromedp.SetValue(`#assign-rad-modal select[name="radiologist_id"]`, "rad_limited", chromedp.ByQuery),
			chromedp.Click(`#assign-rad-modal button[type="submit"]`, chromedp.ByQuery),
			chromedp.WaitVisible(fmt.Sprintf(`//tr[td[contains(text(), "%s")]]//span[contains(text(), "rad_limited")]`, shiftName), chromedp.BySearch),
		)
		if err != nil {
			t.Fatalf("Failed setup for OverAssignment: %v", err)
		}

		var result1 string
		err = chromedp.Run(ctx,
			chromedp.Evaluate(`
				var token = document.querySelector('input[name="csrf_token"]').value;
				fetch('/api/simulate', {
					method: 'POST',
					headers: {
						'Content-Type': 'application/x-www-form-urlencoded',
						'X-CSRF-Token': token
					},
					body: 'study_id=STUDY_1&modality=XRAY'
				}).then(r => r.text())
			`, &result1),
		)
		if err != nil {
			t.Fatalf("Failed simulation 1: %v", err)
		}
		if !strings.Contains(result1, "Assigned to rad_limited") {
			t.Errorf("Expected assignment to rad_limited, got: %s", result1)
		}

		var resMap map[string]interface{}
		err = chromedp.Run(ctx,
			chromedp.Evaluate(`
				var token = document.querySelector('input[name="csrf_token"]').value;
				fetch('/api/simulate', {
					method: 'POST',
					headers: {
						'Content-Type': 'application/x-www-form-urlencoded',
						'X-CSRF-Token': token
					},
					body: 'study_id=STUDY_2&modality=XRAY'
				}).then(r => r.text().then(t => ({status: r.status, text: t})))
			`, &resMap),
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
