// Background service worker

// Listen for installation
chrome.runtime.onInstalled.addListener((details) => {
    if (details.reason === 'install') {
        console.log('[PFC] Extension installed');
        // Open welcome page or onboarding if needed
    }
});

// Handle messages from content scripts or popup
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    if (request.type === 'PORTFOLIO_DETECTED') {
        console.log('[PFC] Portfolio detected:', request.data);

        // Update badge to show number of tickers
        const count = request.data.tickers.length;
        if (count > 0) {
            chrome.action.setBadgeText({ text: count.toString() });
            chrome.action.setBadgeBackgroundColor({ color: '#2563eb' });
        } else {
            chrome.action.setBadgeText({ text: '' });
        }
    }
});

// Alarm for periodic cache cleanup (optional)
chrome.alarms.create('cleanupCache', { periodInMinutes: 60 });

chrome.alarms.onAlarm.addListener((alarm) => {
    if (alarm.name === 'cleanupCache') {
        chrome.storage.local.get(['lastForecast'], (result) => {
            if (result.lastForecast) {
                const age = Date.now() - new Date(result.lastForecast.generatedAt).getTime();
                if (age > 24 * 60 * 60 * 1000) { // Clear if older than 24 hours
                    chrome.storage.local.remove('lastForecast');
                }
            }
        });
    }
});
