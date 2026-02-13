from playwright.sync_api import sync_playwright
import time

def run(playwright):
    browser = playwright.chromium.launch(headless=True)
    context = browser.new_context(viewport={'width': 1280, 'height': 800})
    page = context.new_page()

    # 1. Shifts Page - Edit Modal
    print("Navigating to Shifts...")
    page.goto("http://localhost:8080/shifts")

    # Click edit on the first shift
    print("Opening Edit Shift Modal...")
    try:
        # Find button that calls openEditShiftModal
        page.locator("button[onclick*='openEditShiftModal']").first.click()
        time.sleep(1) # wait for animation/modal open
        page.screenshot(path="verify_shifts_edit.png")
        print("Captured verify_shifts_edit.png")
    except Exception as e:
        print(f"Failed shifts: {e}")
        page.screenshot(path="verify_shifts_fail.png")

    # 2. Procedures Page - Edit Modal
    print("Navigating to Procedures...")
    page.goto("http://localhost:8080/procedures")
    print("Opening Edit Procedure Modal...")
    try:
        page.locator("button[onclick*='openEditProcedure']").first.click()
        time.sleep(1)
        page.screenshot(path="verify_procedures_edit.png")
        print("Captured verify_procedures_edit.png")
    except Exception as e:
        print(f"Failed procedures: {e}")
        page.screenshot(path="verify_procedures_fail.png")

    # 3. Calendar Page - Roster Modal
    print("Navigating to Calendar...")
    page.goto("http://localhost:8080/calendar")
    print("Opening Roster Modal...")
    try:
        # Click on the first chip in Unfilled Shifts or Grid
        # Using attribute selector for onclick containing openRosterModal
        # Note: we used single quotes in HTML for onclick value
        page.locator("div[onclick*='openRosterModal']").first.click()
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
