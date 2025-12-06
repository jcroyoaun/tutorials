/**
 * Posadev Service Discovery - Frontend Application
 * Fetches and displays Kubernetes service discovery data
 */

// Configuration
const CONFIG = {
    dataUrl: 'discovery.json',
    refreshInterval: 30000, // 30 seconds
};

// DOM Elements
const elements = {
    status: document.getElementById('status'),
    lastUpdated: document.getElementById('lastUpdated'),
    totalNamespaces: document.getElementById('totalNamespaces'),
    totalServices: document.getElementById('totalServices'),
    uniqueVersions: document.getElementById('uniqueVersions'),
    namespacesGrid: document.getElementById('namespacesGrid'),
    loadingState: document.getElementById('loadingState'),
    errorState: document.getElementById('errorState'),
    errorMessage: document.getElementById('errorMessage'),
    statsRow: document.getElementById('statsRow'),
};

// State
let refreshTimer = null;

/**
 * Format timestamp to human-readable string
 */
function formatTime(date) {
    return date.toLocaleTimeString('en-US', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
    });
}

/**
 * Update status indicator
 */
function setStatus(status, message) {
    elements.status.className = `status-indicator ${status}`;
    elements.status.querySelector('.status-text').textContent = message;
}

/**
 * Calculate statistics from data
 */
function calculateStats(data) {
    const namespaces = Object.keys(data);
    const allServices = namespaces.flatMap(ns => data[ns]);
    const versions = new Set(allServices.map(s => s.version));
    
    return {
        namespaces: namespaces.length,
        services: allServices.length,
        versions: versions.size,
    };
}

/**
 * Create namespace icon SVG
 */
function createNamespaceIcon() {
    return `
        <svg class="namespace-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
            <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
            <line x1="12" y1="22.08" x2="12" y2="12"/>
        </svg>
    `;
}

/**
 * Render a single service item
 */
function renderService(service) {
    const isLatest = service.version.toLowerCase() === 'latest';
    const versionClass = isLatest ? 'service-version latest' : 'service-version';
    
    return `
        <div class="service-item">
            <div class="service-info">
                <div class="service-dot"></div>
                <span class="service-name">${escapeHtml(service.name)}</span>
            </div>
            <span class="${versionClass}">${escapeHtml(service.version)}</span>
        </div>
    `;
}

/**
 * Render a namespace card
 */
function renderNamespace(namespace, services) {
    const servicesList = services.map(renderService).join('');
    
    return `
        <div class="namespace-card">
            <div class="namespace-header">
                <div class="namespace-name">
                    ${createNamespaceIcon()}
                    ${escapeHtml(namespace)}
                </div>
                <span class="service-count">${services.length} service${services.length !== 1 ? 's' : ''}</span>
            </div>
            <div class="services-list">
                ${servicesList}
            </div>
        </div>
    `;
}

/**
 * Escape HTML to prevent XSS
 */
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

/**
 * Render the complete UI
 */
function renderData(data) {
    // Calculate and update stats
    const stats = calculateStats(data);
    elements.totalNamespaces.textContent = stats.namespaces;
    elements.totalServices.textContent = stats.services;
    elements.uniqueVersions.textContent = stats.versions;
    
    // Sort namespaces alphabetically
    const sortedNamespaces = Object.keys(data).sort();
    
    // Render namespace cards
    const cards = sortedNamespaces.map(ns => renderNamespace(ns, data[ns])).join('');
    elements.namespacesGrid.innerHTML = cards;
    
    // Show content, hide loading
    elements.loadingState.style.display = 'none';
    elements.errorState.style.display = 'none';
    elements.statsRow.style.display = 'grid';
    elements.namespacesGrid.style.display = 'grid';
}

/**
 * Show error state
 */
function showError(message) {
    elements.loadingState.style.display = 'none';
    elements.statsRow.style.display = 'none';
    elements.namespacesGrid.style.display = 'none';
    elements.errorState.style.display = 'block';
    elements.errorMessage.textContent = message;
    setStatus('error', 'Disconnected');
}

/**
 * Fetch and load data
 */
async function loadData() {
    try {
        // Show loading on first load
        if (elements.namespacesGrid.innerHTML === '') {
            elements.loadingState.style.display = 'flex';
        }
        
        setStatus('', 'Fetching...');
        
        const response = await fetch(CONFIG.dataUrl, {
            cache: 'no-store',
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const data = await response.json();
        
        renderData(data);
        setStatus('connected', 'Live');
        elements.lastUpdated.textContent = `Updated: ${formatTime(new Date())}`;
        
    } catch (error) {
        console.error('Failed to load data:', error);
        showError(error.message || 'Failed to fetch service data');
    }
}

/**
 * Start auto-refresh
 */
function startAutoRefresh() {
    if (refreshTimer) {
        clearInterval(refreshTimer);
    }
    refreshTimer = setInterval(loadData, CONFIG.refreshInterval);
}

/**
 * Initialize the application
 */
function init() {
    // Initial load
    loadData();
    
    // Start auto-refresh
    startAutoRefresh();
    
    // Refresh on visibility change (when tab becomes visible)
    document.addEventListener('visibilitychange', () => {
        if (document.visibilityState === 'visible') {
            loadData();
        }
    });
}

// Start the app
init();

