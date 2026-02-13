from playwright.sync_api import sync_playwright
import time

def run(playwright):
    browser = playwright.chromium.launch(headless=True)
    context = browser.new_context(viewport={'width': 1280, 'height': 800})
    page = context.new_page()

    page.on("console", lambda msg: print(f"CONSOLE: {msg.text}"))
    page.on("pageerror", lambda exc: print(f"PAGE ERROR: {exc}"))

    # 3. Calendar Page - Roster Modal
    print("Navigating to Calendar...")
    page.goto("http://localhost:8080/calendar")
    print("Opening Roster Modal...")
    try:
        page.locator("div[onclick*='openRosterModal']").first.click()
        print("Clicked roster chip.")

        # Check if modal has 'active' class
        # BeerCSS adds 'active' class to dialog
        page.wait_for_selector("dialog#manage-roster-modal.active", state="visible", timeout=5000)

        time.sleep(1)
        page.screenshot(path="verify_calendar_roster.png")
        print("Captured verify_calendar_roster.png")
    except Exception as e:
        print(f"Failed calendar: {e}")
        page.screenshot(path="verify_calendar_fail.png")

    browser.close()

if __name__ == "__main__":
    with sync_playwright() as playwright:
        run(playwright)
