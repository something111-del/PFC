// API Configuration
const API_URL = 'https://pfc-go-api-z6xxcgcfla-uc.a.run.app/v1';

document.addEventListener('DOMContentLoaded', async () => {
    // Initialize UI
    const views = {
        loading: document.getElementById('loading-view'),
        empty: document.getElementById('empty-view'),
        results: document.getElementById('results-view'),
        error: document.getElementById('error-view')
    };

    // Check system status
    checkSystemStatus();

    // Get current tab URL to detect if we're on a new stock page
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    const currentUrl = tab?.url || '';

    // Load cached data
    const cached = await chrome.storage.local.get(['portfolioData', 'lastForecast', 'lastUrl']);

    // Clear cache if URL changed (navigated to different stock)
    if (cached.lastUrl && cached.lastUrl !== currentUrl) {
        console.log('[PFC] URL changed, clearing cache');
        await chrome.storage.local.remove(['lastForecast', 'portfolioData']);
        cached.lastForecast = null;
        cached.portfolioData = null;
    }

    // Store current URL
    chrome.storage.local.set({ lastUrl: currentUrl });

    if (cached.lastForecast && isFresh(cached.lastForecast.generatedAt)) {
        showResults(cached.lastForecast);
    } else if (cached.portfolioData) {
        fetchForecast(cached.portfolioData.tickers, cached.portfolioData.portfolio);
    } else {
        // Try to scan current tab
        scanCurrentTab();
    }

    // Event Listeners
    document.getElementById('btn-scan').addEventListener('click', () => {
        chrome.storage.local.remove(['lastForecast']); // Clear cache on manual scan
        scanCurrentTab();
    });

    document.getElementById('btn-retry').addEventListener('click', () => {
        chrome.storage.local.remove(['lastForecast']); // Clear cache on retry
        scanCurrentTab();
    });

    document.getElementById('btn-add').addEventListener('click', () => {
        const input = document.getElementById('manual-ticker');
        const ticker = input.value.trim().toUpperCase();
        if (ticker) {
            chrome.storage.local.remove(['lastForecast']); // Clear cache for new ticker
            fetchForecast([ticker]);
            input.value = '';
        }
    });
});

async function scanCurrentTab() {
    showView('loading');

    try {
        const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });

        if (!tab) throw new Error('No active tab');

        // Send message to content script
        chrome.tabs.sendMessage(tab.id, { action: 'extractTickers' }, (response) => {
            if (chrome.runtime.lastError) {
                // Content script might not be loaded (e.g. restricted page)
                showView('empty');
                return;
            }

            if (response && response.tickers && response.tickers.length > 0) {
                fetchForecast(response.tickers, response.portfolio);
            } else {
                showView('empty');
            }
        });
    } catch (err) {
        console.error('Scan failed:', err);
        showView('empty');
    }
}

async function fetchForecast(tickers, portfolio = [], currentPrice = 0) {
    showView('loading');

    try {
        // Call Go API
        const response = await fetch(`${API_URL}/forecast`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ tickers, portfolio })
        });

        if (!response.ok) throw new Error('API request failed');

        const data = await response.json();

        // If we have a detected price from the page, override the API's current value if it's 0
        if (currentPrice > 0 && (data.currentValue === 0 || !data.currentValue)) {
            data.currentValue = currentPrice;
            // Also adjust expected value relative to the detected price if API returned 0
            if (data.expectedValue === 0) {
                data.expectedValue = currentPrice; // Flat line fallback
                data.percentiles.p5 = currentPrice * 0.95;
                data.percentiles.p50 = currentPrice;
                data.percentiles.p95 = currentPrice * 1.05;
            }
        }

        // Cache result
        chrome.storage.local.set({ lastForecast: data });

        showResults(data);
    } catch (err) {
        console.error('Forecast failed:', err);
        document.getElementById('error-message').textContent = 'Failed to generate forecast. Please try again.';
        showView('error');
    }
}

function showResults(data) {
    showView('results');

    // If no portfolio (currentValue is 0), use the first ticker's data for display
    let displayCurrentValue = data.currentValue;
    let displayExpectedValue = data.expectedValue;
    let displayPercentiles = data.percentiles;

    if (displayCurrentValue === 0 && data.tickers && data.tickers.length > 0) {
        // Sum up all tickers' current prices (assume 1 share each for visualization)
        displayCurrentValue = data.tickers.reduce((sum, t) => sum + t.currentPrice, 0);
        displayExpectedValue = data.tickers.reduce((sum, t) => sum + t.forecast.p50, 0);
        displayPercentiles = {
            p5: data.tickers.reduce((sum, t) => sum + t.forecast.p5, 0),
            p50: data.tickers.reduce((sum, t) => sum + t.forecast.p50, 0),
            p95: data.tickers.reduce((sum, t) => sum + t.forecast.p95, 0)
        };
    }

    // Update Summary
    document.getElementById('current-value').textContent = formatCurrency(displayCurrentValue);
    document.getElementById('expected-value').textContent = formatCurrency(displayExpectedValue);

    const riskBadge = document.getElementById('risk-level');
    riskBadge.textContent = data.risk.toUpperCase();
    riskBadge.className = `value risk-badge risk-${data.risk}`;

    // Render Chart with corrected values
    renderChart({
        ...data,
        currentValue: displayCurrentValue,
        expectedValue: displayExpectedValue,
        percentiles: displayPercentiles
    });

    // Render Tickers List
    const list = document.getElementById('tickers-list');
    list.innerHTML = '';

    data.tickers.forEach(ticker => {
        const el = document.createElement('div');
        el.className = 'ticker-item';

        // Use ticker current price if available, otherwise fallback
        const price = ticker.currentPrice > 0 ? ticker.currentPrice : displayCurrentValue;

        const change = ((ticker.forecast.p50 - price) / price) * 100;
        const changeClass = change >= 0 ? 'positive' : 'negative';

        el.innerHTML = `
            <div class="ticker-info">
                <span class="ticker-symbol">${ticker.symbol}</span>
                <span class="ticker-price">${formatCurrency(price)}</span>
            </div>
            <div class="ticker-forecast">
                <div class="forecast-change ${changeClass}">
                    ${change >= 0 ? '+' : ''}${change.toFixed(2)}%
                </div>
                <div class="label">Exp. ${formatCurrency(ticker.forecast.p50)}</div>
            </div>
        `;
        list.appendChild(el);
    });

    document.getElementById('last-updated').textContent = `Updated: ${new Date(data.generatedAt).toLocaleTimeString()}`;
}

let chartInstance = null;

function renderChart(data) {
    const ctx = document.getElementById('forecast-chart').getContext('2d');

    if (chartInstance) chartInstance.destroy();

    // Generate hourly labels
    const labels = Array.from({ length: 24 }, (_, i) => `${i + 1}h`);

    // Use the actual current value from data (which we fixed above)
    const currentVal = data.currentValue;

    // If expected value is 0, it means API failed to predict, so just show flat line
    const expectedVal = data.currentValue > 0 && data.expectedValue === 0 ? data.currentValue : data.expectedValue;
    const p95 = data.currentValue > 0 && data.percentiles.p95 === 0 ? currentVal * 1.05 : data.percentiles.p95;
    const p5 = data.currentValue > 0 && data.percentiles.p5 === 0 ? currentVal * 0.95 : data.percentiles.p5;

    chartInstance = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [{
                label: 'Expected',
                data: generatePath(currentVal, expectedVal),
                borderColor: '#2563eb',
                borderWidth: 2,
                tension: 0.4,
                pointRadius: 0
            }, {
                label: 'Best Case (95%)',
                data: generatePath(currentVal, p95),
                borderColor: '#22c55e',
                borderWidth: 1,
                borderDash: [5, 5],
                tension: 0.4,
                pointRadius: 0
            }, {
                label: 'Worst Case (5%)',
                data: generatePath(currentVal, p5),
                borderColor: '#ef4444',
                borderWidth: 1,
                borderDash: [5, 5],
                tension: 0.4,
                pointRadius: 0
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { display: false },
                tooltip: { mode: 'index', intersect: false }
            },
            scales: {
                x: { display: false },
                y: {
                    display: true,
                    grid: { color: 'rgba(255, 255, 255, 0.1)' },
                    ticks: { color: '#94a3b8' }
                }
            }
        }
    });
}

// Helper: Generate a smooth path between start and end (linear interpolation for demo)
function generatePath(start, end) {
    const points = [];
    for (let i = 0; i <= 24; i++) {
        points.push(start + (end - start) * (i / 24));
    }
    return points;
}

function showView(viewName) {
    document.querySelectorAll('.view').forEach(el => el.classList.add('hidden'));
    document.getElementById(`${viewName}-view`).classList.remove('hidden');
}

function formatCurrency(val) {
    return new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: 'USD'
    }).format(val);
}

function isFresh(timestamp) {
    return (new Date() - new Date(timestamp)) < 1000 * 60 * 60; // 1 hour freshness
}

async function checkSystemStatus() {
    try {
        const res = await fetch(`${API_URL}/health`);
        if (res.ok) {
            document.getElementById('status-indicator').classList.add('online');
        }
    } catch (e) {
        // Offline
    }
}
