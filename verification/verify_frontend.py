from playwright.sync_api import sync_playwright

def run(playwright):
    browser = playwright.chromium.launch(headless=True)
    context = browser.new_context()
    page = context.new_page()

    # Visit Dashboard
    print("Visiting Dashboard...")
    page.goto("http://localhost:8080/")
    page.wait_for_selector("text=System Status")
    page.screenshot(path="verification/dashboard.png")
    print("Dashboard screenshot saved.")

    # Visit Rules
    print("Visiting Rules...")
    page.goto("http://localhost:8080/rules")
    page.wait_for_selector("text=Assignment Rules")
    page.screenshot(path="verification/rules.png")
    print("Rules screenshot saved.")

    browser.close()

with sync_playwright() as playwright:
    run(playwright)
