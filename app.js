// Global State
let apiKey = '';
let proxyUrl = 'https://corsproxy.io/?';
let useProxy = true;
let currentRecords = [];
let extractedSubdomains = new Set();
let extractedIPs = new Set();
let ipStats = {};

// Initialization
document.addEventListener('DOMContentLoaded', () => {
    loadConfig();
    switchTab('search');
});

// --- Configuration ---

function toggleConfig() {
    const panel = document.getElementById('configPanel');
    panel.classList.toggle('hidden');
}

function toggleProxyInput() {
    const mode = document.querySelector('input[name="connectionMode"]:checked').value;
    const group = document.getElementById('proxyInputGroup');
    if (mode === 'proxy') {
        group.classList.remove('opacity-50', 'pointer-events-none');
        document.getElementById('proxyUrl').disabled = false;
        useProxy = true;
    } else {
        group.classList.add('opacity-50', 'pointer-events-none');
        document.getElementById('proxyUrl').disabled = true;
        useProxy = false;
    }
}

function saveConfig() {
    apiKey = document.getElementById('apiKey').value.trim();
    proxyUrl = document.getElementById('proxyUrl').value.trim();
    const mode = document.querySelector('input[name="connectionMode"]:checked').value;
    useProxy = (mode === 'proxy');

    localStorage.setItem('rapiddns_apiKey', apiKey);
    localStorage.setItem('rapiddns_proxyUrl', proxyUrl);
    localStorage.setItem('rapiddns_useProxy', useProxy);

    toggleConfig();
    alert('Configuration saved!');
}

function loadConfig() {
    const savedKey = localStorage.getItem('rapiddns_apiKey');
    const savedProxy = localStorage.getItem('rapiddns_proxyUrl');
    const savedMode = localStorage.getItem('rapiddns_useProxy');

    if (savedKey) {
        apiKey = savedKey;
        document.getElementById('apiKey').value = apiKey;
    }
    if (savedProxy) {
        proxyUrl = savedProxy;
        document.getElementById('proxyUrl').value = proxyUrl;
    }
    if (savedMode !== null) {
        useProxy = (savedMode === 'true');
        const radios = document.getElementsByName('connectionMode');
        for (const r of radios) {
            if (useProxy && r.value === 'proxy') r.checked = true;
            if (!useProxy && r.value === 'direct') r.checked = true;
        }
        toggleProxyInput();
    }
}

// --- UI Navigation ---

function switchTab(tab) {
    // Hide all
    ['search', 'advanced', 'export'].forEach(t => {
        document.getElementById(`content-${t}`).classList.add('hidden');
        document.getElementById(`tab-${t}`).classList.remove('border-blue-600', 'text-blue-600');
        document.getElementById(`tab-${t}`).classList.add('text-gray-500');
    });

    // Show selected
    document.getElementById(`content-${tab}`).classList.remove('hidden');
    document.getElementById(`tab-${tab}`).classList.add('border-blue-600', 'text-blue-600');
    document.getElementById(`tab-${tab}`).classList.remove('text-gray-500');
}

// --- API Logic ---

const BASE_URL = 'https://rapiddns.io/api';

async function fetchAPI(endpoint, params = {}) {
    const url = new URL(BASE_URL + endpoint);

    // Add params
    Object.keys(params).forEach(key => url.searchParams.append(key, params[key]));

    let fetchUrl = url.toString();
    const headers = {
        'Accept': 'application/json'
    };

    if (apiKey) {
        headers['X-API-KEY'] = apiKey;
    }

    if (useProxy) {
        // Encode the target URL component if the proxy expects it,
        // OR just append if the proxy works that way.
        // Standard corsproxy.io usage: https://corsproxy.io/?https%3A%2F%2Fapi...
        fetchUrl = proxyUrl + encodeURIComponent(fetchUrl);
    }

    try {
        const response = await fetch(fetchUrl, { headers });
        if (!response.ok) {
            if (response.status === 401 || response.status === 403) {
                throw new Error("Invalid API Key or Plan limit reached.");
            }
            throw new Error(`API Error: ${response.statusText}`);
        }
        const data = await response.json();
        return data;
    } catch (error) {
        console.error("Fetch error:", error);
        throw error;
    }
}

// --- Search Logic ---

async function performSearch() {
    const keyword = document.getElementById('searchKeyword').value.trim();
    if (!keyword) return alert("Please enter a keyword");

    const type = document.getElementById('searchType').value;
    const max = parseInt(document.getElementById('searchMax').value) || 1000;

    resetResults();
    setLoading(true);

    try {
        await runSearchLoop(keyword, type, max, 'search');
    } catch (e) {
        alert("Search failed: " + e.message);
    } finally {
        setLoading(false);
    }
}

async function performAdvancedQuery() {
    const query = document.getElementById('advancedQuery').value.trim();
    if (!query) return alert("Please enter a query");

    // For advanced query, we usually want fewer pages or handle it similarly?
    // The CLI treats advanced query separately but structure is similar.
    // Let's reuse the loop but with a different endpoint.

    resetResults();
    setLoading(true);

    try {
        await runSearchLoop(query, '', 1000, 'advanced');
    } catch (e) {
        alert("Query failed: " + e.message);
    } finally {
        setLoading(false);
    }
}

// Generic loop for pagination
async function runSearchLoop(input, type, maxRecords, mode) {
    let page = 1;
    let pageSize = 100; // API default/max usually
    let keepFetching = true;

    while (keepFetching) {
        let resultData = null;

        // Determine which API call to make
        if (mode === 'search') {
            const params = { page, pagesize: pageSize };
            if (type) params.search_type = type;
            const resp = await fetchAPI(`/search/${input}`, params);
            resultData = parseResponse(resp);
        } else {
            // Advanced
            const params = { page, pagesize: pageSize };
            const resp = await fetchAPI(`/search/query/${input}`, params);
            resultData = parseResponse(resp);
        }

        if (!resultData) break; // Error or empty

        const records = resultData.data || resultData.result || [];

        if (records.length === 0) {
            break;
        }

        // Append to global
        currentRecords = currentRecords.concat(records);

        // Update UI
        renderTable(records);
        updateStats();

        // Check limits
        if (currentRecords.length >= maxRecords) {
            keepFetching = false;
        }

        // Check if last page
        if (records.length < pageSize) {
            keepFetching = false;
        }

        page++;

        // Small delay to be nice?
        await new Promise(r => setTimeout(r, 100));
    }
}

// Helper to handle weird API response structures
function parseResponse(resp) {
    // The API might wrap data in 'message' string sometimes or 'data' field
    // Go code logic:
    // 1. Check resp.message (string) -> JSON parse -> check data
    // 2. Check resp.data (object or string)

    let searchData = null;

    if (resp.message && typeof resp.message === 'string') {
        try {
            const parsed = JSON.parse(resp.message);
            if (parsed.data || parsed.result) return parsed;
        } catch (e) {}
    }

    if (resp.data) {
        if (typeof resp.data === 'string') {
            try {
                const parsed = JSON.parse(resp.data);
                return parsed;
            } catch (e) {}
        } else {
            return resp.data; // Already object
        }
    }

    // Direct access if structure is flat?
    if (resp.result || (resp.data && Array.isArray(resp.data))) {
        return resp;
    }

    return null;
}

function resetResults() {
    currentRecords = [];
    extractedSubdomains.clear();
    extractedIPs.clear();
    ipStats = {};

    document.getElementById('resultsTableBody').innerHTML = '';
    document.getElementById('resultCount').innerText = '0';
    document.getElementById('subCount').innerText = '0';
    document.getElementById('ipCount').innerText = '0';
    document.getElementById('extractedSubdomains').value = '';
    document.getElementById('extractedIPs').value = '';
    document.getElementById('extractedStats').value = '';

    document.getElementById('resultsArea').classList.remove('hidden');
}

function setLoading(isLoading) {
    const btn = document.getElementById('searchSpinner');
    const txt = document.getElementById('searchBtnText');
    if (isLoading) {
        btn.classList.remove('hidden');
        txt.innerText = 'Searching...';
    } else {
        btn.classList.add('hidden');
        txt.innerText = 'Search';
    }
}

// --- Data Processing & Rendering ---

function renderTable(newRecords) {
    const tbody = document.getElementById('resultsTableBody');
    const extractSubs = document.getElementById('autoExtractSubdomains').checked;
    const extractIPsOpt = document.getElementById('autoExtractIPs').checked;

    newRecords.forEach(r => {
        // UI
        const row = document.createElement('tr');
        row.innerHTML = `
            <td class="px-6 py-4 whitespace-nowrap text-gray-900">${r.subdomain || '-'}</td>
            <td class="px-6 py-4 whitespace-nowrap text-gray-500"><span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-blue-100 text-blue-800">${r.type}</span></td>
            <td class="px-6 py-4 whitespace-nowrap text-gray-500">${r.value}</td>
            <td class="px-6 py-4 whitespace-nowrap text-gray-500">${r.date || '-'}</td>
        `;
        tbody.appendChild(row);

        // Extraction Logic
        if (extractSubs && r.subdomain) {
            extractedSubdomains.add(r.subdomain);
        }
        if (extractIPsOpt && r.type === 'A' || r.type === 'AAAA' || (r.value && r.value.match(/^[0-9a-fA-F:.]+$/))) {
             // Simple IP check
             // Note: r.type might be 'A' but value is IP.
             // If type is not A/AAAA, but value looks like IP (e.g. from same_ip search), we use it.
             if (isValidIP(r.value)) {
                 if (!extractedIPs.has(r.value)) {
                     extractedIPs.add(r.value);
                     calcIpStats(r.value);
                 }
             }
        }
    });

    document.getElementById('resultCount').innerText = currentRecords.length;
}

function updateStats() {
    // Update Subdomains Text
    const subs = Array.from(extractedSubdomains).sort();
    document.getElementById('extractedSubdomains').value = subs.join('\n');
    document.getElementById('subCount').innerText = subs.length;

    // Update IPs Text
    const ips = Array.from(extractedIPs).sort(sortIPs);
    document.getElementById('extractedIPs').value = ips.join('\n');
    document.getElementById('ipCount').innerText = ips.length;

    // Update Stats Text
    const statsLines = Object.keys(ipStats).sort().map(k => `${k}: ${ipStats[k]} IPs`);
    document.getElementById('extractedStats').value = statsLines.join('\n');
}

function isValidIP(ip) {
    // Very basic check
    return ip.includes('.') || ip.includes(':');
}

function calcIpStats(ip) {
    let subnet = '';
    if (ip.includes('.')) {
        // IPv4 /24
        const parts = ip.split('.');
        if (parts.length === 4) {
            subnet = parts.slice(0, 3).join('.') + '.0/24';
        }
    } else if (ip.includes(':')) {
        // IPv6 /64
        // This is tricky without a library.
        // We will try to take the first 4 blocks.
        // If compressed :: is at start or middle, expansion is needed for accurate /64.
        // For now, naive approach: exact match of prefix if full?
        // Let's just group by "first half" if possible or ignore if too complex.
        // Actually, let's just use the full IP as stats key if we can't parse, or try a simple heuristic.
        // Heuristic: Split by :, if we have at least 4 parts before ::, take them.
        // Better: just skip IPv6 stats aggregation to avoid bad data, or count them individually?
        // Let's try to just output "IPv6" bucket or use the input as is?
        // No, let's try a simple aggregation:
        // 2001:db8:1:2:3:4:5:6 -> 2001:db8:1:2::/64
        // If we split by ':', take first 4.
        const parts = ip.split(':');
        if (parts.length >= 4) {
             subnet = parts.slice(0, 4).join(':') + '::/64';
        } else {
            subnet = 'IPv6-Other';
        }
    }

    if (subnet) {
        ipStats[subnet] = (ipStats[subnet] || 0) + 1;
    }
}

function sortIPs(a, b) {
    // specialized sort? or just string sort
    return a.localeCompare(b, undefined, { numeric: true });
}

// --- Export Logic ---

async function startExport() {
    const input = document.getElementById('exportInput').value;
    const type = document.getElementById('exportType').value;

    if (!input) return alert("Enter export input");

    // POST /api/export-data
    // Body: query_type, query_input, max_results=0, compress=true

    // Note: Fetch with body
    const url = new URL(BASE_URL + '/export-data');
    let fetchUrl = url.toString();

    if (useProxy) {
        fetchUrl = proxyUrl + encodeURIComponent(fetchUrl);
    }

    const payload = {
        query_type: type,
        query_input: input,
        max_results: 0,
        compress: true
    };

    try {
        const resp = await fetch(fetchUrl, {
            method: 'POST',
            headers: {
                'X-API-KEY': apiKey,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(payload)
        });

        const data = await resp.json();
        const parsed = parseResponse(data);

        if (parsed && parsed.export_id) {
            alert(`Export Started! ID: ${parsed.export_id}. Check status tab.`);
            document.getElementById('exportTaskId').value = parsed.export_id;
            switchTab('export'); // Stay on export tab
        } else {
            alert("Failed to start export: " + (data.msg || "Unknown error"));
        }
    } catch (e) {
        alert("Export Error: " + e.message);
    }
}

async function checkExportStatus() {
    const id = document.getElementById('exportTaskId').value;
    if (!id) return alert("Enter Task ID");

    const resultDiv = document.getElementById('exportStatusResult');
    resultDiv.classList.remove('hidden');
    document.getElementById('statusText').innerText = "Checking...";

    try {
        const resp = await fetchAPI(`/export-data/${id}`);
        const data = parseResponse(resp);

        if (data) {
            document.getElementById('statusText').innerText = data.status;
            document.getElementById('progressText').innerText = (data.progress_percent || 0) + '%';

            if (data.status === 'completed' && data.download_url) {
                const link = document.getElementById('downloadLink');
                link.href = data.download_url;
                link.classList.remove('hidden');
                link.innerText = "Download Result (ZIP)";
            }
        }
    } catch (e) {
        document.getElementById('statusText').innerText = "Error: " + e.message;
    }
}


// --- Download Results ---

function downloadResults(format) {
    if (currentRecords.length === 0) return alert("No results to download");

    let content = '';
    let mime = 'text/plain';
    let filename = `rapiddns_results.${format}`;

    if (format === 'json') {
        content = JSON.stringify(currentRecords, null, 2);
        mime = 'application/json';
    } else if (format === 'csv') {
        mime = 'text/csv';
        // Header
        const headers = ['Subdomain', 'Type', 'Value', 'Date'];
        content = headers.join(',') + '\n';
        // Rows
        content += currentRecords.map(r => {
            return [
                `"${r.subdomain || ''}"`,
                `"${r.type || ''}"`,
                `"${r.value || ''}"`,
                `"${r.date || ''}"`
            ].join(',');
        }).join('\n');
    } else {
        // Text
        content = currentRecords.map(r => `${r.subdomain || ''}\t${r.type}\t${r.value}\t${r.date}`).join('\n');
    }

    triggerDownload(content, filename, mime);
}

function downloadExtraction(type) {
    let content = '';
    let filename = '';

    if (type === 'subdomains') {
        content = document.getElementById('extractedSubdomains').value;
        filename = 'subdomains.txt';
    } else if (type === 'ips') {
        const ips = document.getElementById('extractedIPs').value;
        const stats = document.getElementById('extractedStats').value;
        content = "--- IPs ---\n" + ips + "\n\n--- Stats ---\n" + stats;
        filename = 'ips_and_stats.txt';
    }

    if (!content) return alert("Nothing to download");
    triggerDownload(content, filename, 'text/plain');
}

function triggerDownload(content, filename, mime) {
    const blob = new Blob([content], { type: mime });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}
