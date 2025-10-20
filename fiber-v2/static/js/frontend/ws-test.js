// Simple WebSocket test client
const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws';
const url = `${scheme}://${window.location.host}/backend/notifications`;

// ikinci argüman ayarlanırsa hata fırlatıyor.
const socket = new WebSocket(url, 'kullanici');

socket.addEventListener('open', (event) => {
    console.log('[WS] Connected:', url, event);
});

socket.addEventListener('message', (event) => {
    // bu hırremin de blob tipinde geliyor.
    console.log('[WS] Message:', event.data);
});

socket.addEventListener('error', (err) => {
    console.error('[WS] Error:', err);
});

socket.addEventListener('close', (evt) => {
    console.log('[WS] Closed:', evt.code, evt.reason || '');
});





