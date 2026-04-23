function toggleSidebar() {
    const sidebar = document.querySelector('.sidebar');
    sidebar.classList.toggle('collapsed');
    sidebar.addEventListener('transitionend', () => window.dispatchEvent(new Event('resize')), { once: true });
}

// Terminal instance
let term = null;
let socket = null;
let sessionUpdateSocket = null;
let currentSession = null;
let sessions = [];

// Keyboard shortcut handler (shared between document and xterm)
function handleShortcut(e) {
    // Alt+N: new session
    if (e.altKey && e.key === 'n') {
        e.preventDefault();
        e.stopPropagation();
        showCreateModal();
        return true;
    }
    // Alt+ArrowUp/Down: switch between sessions
    if (e.altKey && (e.key === 'ArrowUp' || e.key === 'ArrowDown') && sessions.length > 0) {
        e.preventDefault();
        e.stopPropagation();
        const idx = sessions.findIndex(s => s.Name === currentSession);
        let next;
        if (e.key === 'ArrowUp') {
            next = idx <= 0 ? sessions.length - 1 : idx - 1;
        } else {
            next = idx >= sessions.length - 1 ? 0 : idx + 1;
        }
        connectToSession(sessions[next].Name);
        return true;
    }
    return false;
}

// Initialize session update WebSocket
function initSessionUpdates() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws/sessions`;
    sessionUpdateSocket = new WebSocket(wsUrl);

    sessionUpdateSocket.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            if (data.type === 'update') {
                displaySessions(data.sessions);
            }
        } catch (e) {
            console.error('Failed to parse session update:', e);
        }
    };

    sessionUpdateSocket.onclose = () => {
        // Reconnect after 3 seconds
        console.log('Session update WebSocket closed, reconnecting...');
        setTimeout(initSessionUpdates, 3000);
    };

    sessionUpdateSocket.onerror = (error) => {
        console.error('Session update WebSocket error:', error);
    };
}

// Initialize xterm.js
function initTerminal() {
    console.log('Initializing terminal...');

    // Dispose existing terminal if any
    if (term) {
        term.dispose();
        term = null;
    }

    // Clear terminal container
    const terminalContainer = document.getElementById('terminal');
    terminalContainer.innerHTML = '';

    term = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: 'Consolas, "Courier New", monospace',
        theme: {
            background: '#000000',
            foreground: '#ffffff',
            cursor: '#ffffff',
            black: '#000000',
            red: '#cd3131',
            green: '#0dbc79',
            yellow: '#e5e510',
            blue: '#2472c8',
            magenta: '#bc3fbc',
            cyan: '#11a8cd',
            white: '#e5e5e5',
            brightBlack: '#666666',
            brightRed: '#f14c4c',
            brightGreen: '#23d18b',
            brightYellow: '#f5f543',
            brightBlue: '#3b8eea',
            brightMagenta: '#d670d6',
            brightCyan: '#29b8db',
            brightWhite: '#ffffff',
        },
    });

    term.open(terminalContainer);
    term.attachCustomKeyEventHandler((e) => {
        if (e.type !== 'keydown') return true;
        return !handleShortcut(e);
    });
    term.onData((data) => {
        if (socket && socket.readyState === WebSocket.OPEN) {
            // Encode as UTF-8 bytes, then base64
            const encoder = new TextEncoder();
            const bytes = encoder.encode(data);
            const binaryString = String.fromCharCode.apply(null, bytes);
            const encoded = btoa(binaryString);
            socket.send(encoded);
        }
    });

    // Hide no session message and show terminal
    document.getElementById('noSessionMessage').style.display = 'none';
    document.getElementById('terminal').style.display = 'block';
}

// Connect to tmux session via WebSocket
function connectToSession(sessionName) {
    // Mark the old socket as intentionally closed (don't tear down terminal)
    const oldSocket = socket;
    socket = null;
    if (oldSocket) {
        oldSocket.onclose = null;
        oldSocket.onerror = null;
        oldSocket.close();
    }

    currentSession = sessionName;

    // Initialize terminal if not already done
    if (!term) {
        initTerminal();
    }

    term.clear();

    // Get terminal dimensions
    const cols = term.cols;
    const rows = term.rows;

    // Connect WebSocket with terminal size
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws?session=${encodeURIComponent(sessionName)}&cols=${cols}&rows=${rows}`;
    socket = new WebSocket(wsUrl);
    const activeSocket = socket;

    socket.onopen = () => {
        console.log('Connected to session:', sessionName);
        term.reset();
        term.focus();
        // Re-highlight active session in sidebar
        document.querySelectorAll('.session-item').forEach(item => {
            item.classList.toggle('active', item.dataset.session === sessionName);
        });
    };

    socket.onmessage = (event) => {
        try {
            // Decode base64 and handle UTF-8 properly
            const binaryString = atob(event.data);
            const bytes = new Uint8Array(binaryString.length);
            for (let i = 0; i < binaryString.length; i++) {
                bytes[i] = binaryString.charCodeAt(i);
            }
            const decoder = new TextDecoder('utf-8');
            const text = decoder.decode(bytes);
            term.write(text);
        } catch (e) {
            console.error('Failed to decode message:', e);
        }
    };

    socket.onerror = (error) => {
        console.error('WebSocket error:', error);
        term.write('\r\n\x1b[31mConnection error. Refresh to try again.\x1b[0m\r\n');
    };

    socket.onclose = (event) => {
        console.log('Connection closed:', event.code, event.reason);
        // Only tear down if this socket is still the current one
        if (socket === activeSocket) {
            if (term) {
                term.dispose();
                term = null;
            }
            socket = null;
            currentSession = null;

            // Clear terminal container
            const terminalContainer = document.getElementById('terminal');
            terminalContainer.innerHTML = '';

            // Show no session message
            document.getElementById('noSessionMessage').style.display = 'flex';
            document.getElementById('terminal').style.display = 'none';

            // Refresh sidebar
            loadSessions();
        }
    };

    // Update active state in UI
    document.querySelectorAll('.session-item').forEach(item => {
        item.classList.remove('active');
    });
    document.querySelector(`[data-session="${sessionName}"]`)?.classList.add('active');
}

// Load sessions from API
async function loadSessions() {
    try {
        const response = await fetch('/api/sessions');
        const sessions = await response.json();
        displaySessions(sessions);
    } catch (error) {
        console.error('Failed to load sessions:', error);
    }
}

// Display sessions in sidebar
function displaySessions(sessionList) {
    sessions = sessionList;
    const container = document.getElementById('sessionList');

    if (sessions.length === 0) {
        container.innerHTML = '<div class="empty-state">No active sessions</div>';
        return;
    }

    container.innerHTML = sessions.map(session => `
        <div class="session-item" data-session="${session.Name}" onclick="connectToSession('${session.Name}')">
            <div class="session-name">${escapeHtml(session.Name)}</div>
            <div class="session-info">
                ${session.Windows} window${session.Windows !== '1' ? 's' : ''} • Created ${session.Created || 'recently'}
            </div>
            <div class="session-actions">
                <button class="btn btn-sm" onclick="event.stopPropagation(); connectToSession('${session.Name}')">
                    Connect
                </button>
                <button class="btn btn-danger btn-sm" onclick="event.stopPropagation(); killSession('${session.Name}')">
                    Kill
                </button>
            </div>
        </div>
    `).join('');
}

// Create new session
async function createSession() {
    const name = document.getElementById('sessionName').value.trim();

    if (!name) {
        alert('Please enter a session name');
        return;
    }

    try {
        const response = await fetch('/api/sessions', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name }),
        });

        if (response.ok) {
            hideCreateModal();
            document.getElementById('sessionName').value = '';
            connectToSession(name);
        } else {
            const error = await response.json();
            alert('Failed to create session: ' + error.error);
        }
    } catch (error) {
        console.error('Failed to create session:', error);
        alert('Failed to create session');
    }
}

// Kill session
async function killSession(name) {
    if (!confirm(`Are you sure you want to kill session "${name}"?`)) {
        return;
    }

    try {
        await fetch(`/api/sessions/${encodeURIComponent(name)}`, {
            method: 'DELETE',
        });

        // Clean up terminal if this was the active session
        if (currentSession === name) {
            if (socket) {
                socket.close();
                socket = null;
            }
            if (term) {
                term.dispose();
                term = null;
            }
            currentSession = null;

            // Clear terminal container
            const terminalContainer = document.getElementById('terminal');
            terminalContainer.innerHTML = '';

            // Show no session message
            document.getElementById('noSessionMessage').style.display = 'flex';
            document.getElementById('terminal').style.display = 'none';
        }

        // Refresh session list
        loadSessions();
    } catch (error) {
        console.error('Failed to kill session:', error);
        alert('Failed to kill session');
    }
}

// Modal controls
function showCreateModal() {
    document.getElementById('createModal').classList.add('show');
    document.getElementById('sessionName').focus();
}

function hideCreateModal() {
    document.getElementById('createModal').classList.remove('show');
}

// Utility: Escape HTML
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Initialize session updates on page load
document.addEventListener('DOMContentLoaded', () => {
    initSessionUpdates();
});

// Keyboard shortcuts (only fires when terminal is NOT focused — xterm handles it otherwise)
document.addEventListener('keydown', (e) => {
    if (term && term.element?.contains(document.activeElement)) return;
    if (handleShortcut(e)) return;
    // Enter: submit session creation
    if (e.key === 'Enter' && document.getElementById('createModal').classList.contains('show')) {
        createSession();
    }
    // Escape: close modal
    if (e.key === 'Escape') {
        hideCreateModal();
    }
});

// Load sessions on page load (initial load)
loadSessions();

// Session updates are now pushed via WebSocket (no polling needed!)

