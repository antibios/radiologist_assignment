from playwright.sync_api import sync_playwright
import time

def run(playwright):
    browser = playwright.chromium.launch(headless=True)
    # Desktop View
    context = browser.new_context(viewport={'width': 1280, 'height': 800})
    page = context.new_page()

    print("Navigating to Calendar (Month View)...")
    page.goto("http://localhost:8080/calendar?view=month")
    time.sleep(1)
    page.screenshot(path="verify_calendar_month_desktop.png")
    print("Captured verify_calendar_month_desktop.png")

    # Mobile View
    context_mobile = browser.new_context(viewport={'width': 375, 'height': 667})
    page_mobile = context_mobile.new_page()

    print("Navigating to Calendar (Month View - Mobile)...")
    page_mobile.goto("http://localhost:8080/calendar?view=month")
    time.sleep(1)
    page_mobile.screenshot(path="verify_calendar_month_mobile.png")
    print("Captured verify_calendar_month_mobile.png")

    # Day View
    print("Navigating to Calendar (Day View)...")
    page.goto("http://localhost:8080/calendar?view=day")
    time.sleep(1)
    page.screenshot(path="verify_calendar_day.png")
    print("Captured verify_calendar_day.png")

    browser.close()

if __name__ == "__main__":
    with sync_playwright() as playwright:
        run(playwright)
