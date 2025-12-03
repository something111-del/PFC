// Content script for detecting portfolio tickers on brokerage sites

console.log('[PFC] Content script loaded');

// Platform-specific selectors
const PLATFORM_SELECTORS = {
    'robinhood.com': {
        name: 'Robinhood',
        tickerSelector: '[data-testid="TickerSymbol"], .rh-ticker-symbol, [class*="Symbol"]',
        priceSelector: '[data-testid="Price"], .rh-price',
        sharesSelector: '[data-testid="Quantity"], .rh-quantity'
    },
    'webull.com': {
        name: 'Webull',
        tickerSelector: '.ticker-symbol, [class*="symbol"]',
        priceSelector: '.price, [class*="price"]',
        sharesSelector: '.quantity, [class*="quantity"]'
    },
    'tdameritrade.com': {
        name: 'TD Ameritrade',
        tickerSelector: '.symbol, [data-field="symbol"]',
        priceSelector: '.price, [data-field="price"]',
        sharesSelector: '.quantity, [data-field="quantity"]'
    },
    'etrade.com': {
        name: 'E*TRADE',
        tickerSelector: '.symbol, [id*="symbol"]',
        priceSelector: '.price, [id*="price"]',
        sharesSelector: '.quantity, [id*="quantity"]'
    },
    'fidelity.com': {
        name: 'Fidelity',
        tickerSelector: '.symbol, [data-symbol]',
        priceSelector: '.price, [data-price]',
        sharesSelector: '.quantity, [data-quantity]'
    },
    'schwab.com': {
        name: 'Charles Schwab',
        tickerSelector: '.symbol, [data-label="Symbol"]',
        priceSelector: '.price, [data-label="Price"]',
        sharesSelector: '.quantity, [data-label="Quantity"]'
    }
};

// Detect current platform
function detectPlatform() {
    const hostname = window.location.hostname.replace('www.', '');

    for (const [domain, config] of Object.entries(PLATFORM_SELECTORS)) {
        if (hostname.includes(domain)) {
            return config;
        }
    }

    return null;
}

// Extract tickers from page
function extractTickers() {
    const platform = detectPlatform();
    const url = window.location.href;

    // Strategy 1: Check URL for ticker (e.g. robinhood.com/stocks/AAPL)
    const urlTickerMatch = url.match(/\/stocks\/([A-Z]+)/i) || url.match(/symbol=([A-Z]+)/i);
    let urlTicker = urlTickerMatch ? urlTickerMatch[1].toUpperCase() : null;

    const tickers = new Set();
    const portfolio = [];
    let detectedPrice = 0;

    // If we found a ticker in URL, add it immediately
    if (urlTicker) {
        console.log(`[PFC] Found ticker in URL: ${urlTicker}`);
        tickers.add(urlTicker);

        // Try to find the price for this specific ticker on the page
        if (platform) {
            const priceEl = document.querySelector(platform.priceSelector);
            if (priceEl) {
                const priceText = priceEl.textContent.replace(/[^0-9.]/g, '');
                detectedPrice = parseFloat(priceText);
                console.log(`[PFC] Found price on page: ${detectedPrice}`);
            }
        }
    }

    if (platform) {
        console.log(`[PFC] Scanning DOM for platform: ${platform.name}`);
        const tickerElements = document.querySelectorAll(platform.tickerSelector);

        tickerElements.forEach(element => {
            let ticker = element.textContent.trim().toUpperCase();
            ticker = ticker.replace(/[$\s]/g, ''); // Clean ticker

            if (/^[A-Z]{1,5}$/.test(ticker)) {
                tickers.add(ticker);

                // Try to find associated shares and price
                const row = element.closest('tr, div[class*="row"], li');
                if (row) {
                    const priceEl = row.querySelector(platform.priceSelector);
                    const sharesEl = row.querySelector(platform.sharesSelector);

                    if (sharesEl) {
                        const shares = parseFloat(sharesEl.textContent.replace(/[^0-9.]/g, ''));
                        if (!isNaN(shares) && shares > 0) {
                            portfolio.push({
                                ticker: ticker,
                                shares: shares
                            });
                        }
                    }
                }
            }
        });
    }

    // If we have a URL ticker but no portfolio shares, assume 1 share for visualization
    if (urlTicker && portfolio.length === 0 && detectedPrice > 0) {
        portfolio.push({
            ticker: urlTicker,
            shares: 1,
            price: detectedPrice // Pass this to popup
        });
    }

    const result = {
        platform: platform ? platform.name : 'Unknown',
        tickers: Array.from(tickers),
        portfolio: portfolio.length > 0 ? portfolio : null,
        currentPrice: detectedPrice // Send detected price to popup
    };

    console.log('[PFC] Extracted data:', result);
    return result;
}

// Listen for messages from popup
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    if (request.action === 'extractTickers') {
        const data = extractTickers();
        sendResponse(data);
    }
    return true;
});

// Store extracted data for quick access
let cachedData = null;
let cacheTime = 0;

function updateCache() {
    const now = Date.now();
    if (!cachedData || (now - cacheTime) > 5000) { // Cache for 5 seconds
        cachedData = extractTickers();
        cacheTime = now;

        // Store in chrome.storage for popup access
        chrome.storage.local.set({ portfolioData: cachedData });
    }
}

// Update cache when page loads and on changes
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', updateCache);
} else {
    updateCache();
}

// Watch for DOM changes (portfolio updates)
const observer = new MutationObserver(() => {
    updateCache();
});

observer.observe(document.body, {
    childList: true,
    subtree: true
});

console.log('[PFC] Content script initialized');
