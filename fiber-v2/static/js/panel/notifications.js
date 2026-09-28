/**
 * Notifications WebSocket Handler
 * Connects to /backend/notifications and updates the notification dropdown
 */

class NotificationHandler {
    constructor() {
        this.socket = null;
        this.notificationBtn = document.getElementById('notificationBtn');
        this.notificationDropdown = document.getElementById('notificationDropdown');
        this.notificationList = document.querySelector('.notification-list');
        this.notificationCount = document.querySelector('.notification-count');
        this.isConnected = false;
        
        this.init();
    }

    init() {
        this.connectWebSocket();
        this.setupEventListeners();
    }

    connectWebSocket() {
        const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws';
        const url = `${scheme}://${window.location.host}/backend/notifications`;

        try {
            this.socket = new WebSocket(url, 'kullanici');

            this.socket.addEventListener('open', () => {
                this.isConnected = true;
            });

            this.socket.addEventListener('message', (event) => {
                this.handleNotification(event.data);
            });

            this.socket.addEventListener('error', (error) => {
                console.error('[Notifications] WebSocket error:', error);
                this.isConnected = false;
            });

            this.socket.addEventListener('close', () => {
                this.isConnected = false;
                // Reconnect after 5 seconds
                setTimeout(() => {
                    if (!this.isConnected) {
                        this.connectWebSocket();
                    }
                }, 5000);
            });

        } catch (error) {
            console.error('[Notifications] WebSocket connection failed:', error);
        }
    }

    handleNotification(data) {
        try {
            const message = JSON.parse(data);

            if (message.type === 'new_randevu_talebi') {
                if (typeof window._addNewRandevuRow === 'function') window._addNewRandevuRow(message);
                if (typeof _updateStatCounts === "function") _updateStatCounts(1);
                if (typeof message.notification_message === 'string' && message.notification_message) {
                    this.addNotificationToDropdown({
                        message: message.notification_message,
                        notification_type: message.notification_type,
                        request_link: message.request_link
                    });
                    this.updateNotificationCount();
                    this.addNotificationBadge();
                }
                if (typeof window._pollNewRequests === 'function') {
                    window._pollNewRequests();
                }
                return;
            }

            if (message.type === 'randevu_talebi_status') {
                const row = this.findRequestElement('tr[data-id]', message.rrid);
                const card = this.findRequestElement('.randevu-talebi-card[data-id]', message.rrid);
                const statusLabels = {
                    'yeni': 'Yeni', 'randevu-verildi': 'Randevu Verildi',
                    'randevu-verilemedi': 'Randevu Verilemedi', 'ulasilamadi': 'Ulaşılamadı',
                    'gelmedi': 'Gelmedi', 'hasta-vazgecti': 'Hasta Vazgeçti', 'hasta-arandi': 'Hasta Arandı'
                };
                [row, card].forEach(el => {
                    if (!el) return;
                    const badge = el.querySelector('.status-badge');
                    if (badge) {
                        badge.className = `status-badge status-badge--${message.new_status}`;
                        badge.textContent = statusLabels[message.new_status] || message.new_status;
                    }
                    const btn = el.querySelector('.action-item.toggle-status');
                    if (btn) btn.setAttribute('data-status', message.new_status);
                });
                return;
            }

            if (message.type === 'randevu_talebi_silindi') {
                const row = this.findRequestElement('tr[data-id]', message.rrid);
                const card = this.findRequestElement('.randevu-talebi-card[data-id]', message.rrid);
                if (row) { row.style.opacity = '0'; setTimeout(() => row.remove(), 300); }
                if (typeof _updateStatCounts === "function") _updateStatCounts(-1);
                if (card) { card.style.opacity = '0'; setTimeout(() => card.remove(), 300); }
                return;
            }
            
            if (message.message) {
                this.addNotificationToDropdown(message);
                this.updateNotificationCount();
                this.addNotificationBadge();
            }
        } catch (error) {
            console.error('[Notifications] Error parsing message:', error);
        }
    }

    findRequestElement(selector, rrid) {
        if (typeof rrid !== 'string' && typeof rrid !== 'number') return null;
        return Array.from(document.querySelectorAll(selector))
            .find(element => element.getAttribute('data-id') === String(rrid)) || null;
    }

    addNotificationToDropdown(message) {
        if (!this.notificationList) return;

        // Create notification item
        const notificationItem = document.createElement('div');
        notificationItem.className = 'notification-item unread';
        
        const notificationType = this.getNotificationType(message.notification_type);
        const iconClass = this.getIconClass(notificationType);
        const timeText = this.formatTime(new Date());

        const icon = document.createElement('div');
        icon.className = `notification-icon ${notificationType}`;
        const iconImage = document.createElement('i');
        iconImage.className = `fas ${iconClass}`;
        icon.appendChild(iconImage);

        const content = document.createElement('div');
        content.className = 'notification-content';
        const notificationText = document.createElement('div');
        notificationText.className = 'notification-text';
        notificationText.textContent = message.message;
        const notificationTime = document.createElement('div');
        notificationTime.className = 'notification-time';
        notificationTime.textContent = timeText;
        content.appendChild(notificationText);
        content.appendChild(notificationTime);
        notificationItem.appendChild(icon);
        notificationItem.appendChild(content);

        // Add click handler for notification link
        const requestLink = this.getPanelLink(message.request_link);
        if (requestLink) {
            notificationItem.style.cursor = 'pointer';
            notificationItem.tabIndex = 0;
            notificationItem.setAttribute('role', 'link');
            notificationItem.addEventListener('click', () => {
                window.location.href = requestLink;
            });
            notificationItem.addEventListener('keydown', event => {
                if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    window.location.href = requestLink;
                }
            });
        }

        // Prepend to notification list
        const firstChild = this.notificationList.firstChild;
        if (firstChild) {
            this.notificationList.insertBefore(notificationItem, firstChild);
        } else {
            this.notificationList.appendChild(notificationItem);
        }

        // Remove "Henüz bildirim yok" message if it exists
        const emptyMessage = this.notificationList.querySelector('.notification-item:not(.unread):not([class*="notification-icon"])');
        if (emptyMessage && emptyMessage.textContent.includes('Henüz bildirim yok')) {
            emptyMessage.remove();
        }
    }

    getNotificationType(notificationType) {
        switch (notificationType) {
            case 'success':
            case 'info':
            case 'warning':
            case 'danger':
            case 'error':
                return notificationType;
            default:
                return 'info';
        }
    }

    getPanelLink(link) {
        if (typeof link !== 'string' || !/^\/panel(?:\/|\?|#|$)/.test(link) ||
            /[\u0000-\u001f\u007f\\]/.test(link) || link !== link.trim()) return null;
        try {
            decodeURI(link);
            const url = new URL(link, window.location.origin);
            if (url.origin !== window.location.origin ||
                (url.pathname !== '/panel' && !url.pathname.startsWith('/panel/'))) return null;
            return url.pathname + url.search + url.hash;
        } catch (_) {
            return null;
        }
    }

    updateNotificationCount() {
        if (!this.notificationCount) return;

        const currentCount = parseInt(this.notificationCount.textContent) || 0;
        this.notificationCount.textContent = currentCount + 1;
    }

    addNotificationBadge() {
        if (!this.notificationBtn) return;

        if (!this.notificationBtn.classList.contains('has-notification')) {
            this.notificationBtn.classList.add('has-notification');
        }
    }

    getIconClass(notificationType) {
        const iconMap = {
            'success': 'fa-check-circle',
            'info': 'fa-info-circle',
            'warning': 'fa-exclamation-triangle',
            'danger': 'fa-times-circle',
            'error': 'fa-times-circle'
        };
        return iconMap[notificationType] || 'fa-info-circle';
    }

    formatTime(date) {
        const now = new Date();
        const diff = now - date;
        const minutes = Math.floor(diff / 60000);
        const hours = Math.floor(diff / 3600000);
        const days = Math.floor(diff / 86400000);

        if (minutes < 1) return 'Az önce';
        if (minutes < 60) return `${minutes} dakika önce`;
        if (hours < 24) return `${hours} saat önce`;
        if (days < 7) return `${days} gün önce`;
        
        return date.toLocaleDateString('tr-TR', {
            day: '2-digit',
            month: '2-digit',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    }

    setupEventListeners() {
        // Handle notification button click
        if (this.notificationBtn) {
            this.notificationBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                this.toggleDropdown();
            });
        }

        // Close dropdown when clicking outside
        document.addEventListener('click', (e) => {
            if (!this.notificationBtn?.contains(e.target) && !this.notificationDropdown?.contains(e.target)) {
                this.closeDropdown();
            }
        });

        // Note: Notification badge should only be added by backend or websocket, not when dropdown closes
    }

    toggleDropdown() {
        if (!this.notificationDropdown) return;

        if (this.notificationDropdown.style.display === 'none' || this.notificationDropdown.style.display === '') {
            this.notificationDropdown.style.display = 'block';
        } else {
            this.closeDropdown();
        }
    }

    closeDropdown() {
        if (!this.notificationDropdown) return;

        this.notificationDropdown.style.display = 'none';
        // Note: Don't add notification badge when closing dropdown
        // The badge should only be added by backend or websocket
    }

    // Method to manually add notification (for testing)
    addTestNotification() {
        const testMessage = {
            insert_form: 'randevu',
            message: 'Test bildirimi - Yeni randevu talebi alındı',
            notification_type: 'info',
            request_link: '/panel/randevu-talepleri'
        };
        this.addNotificationToDropdown(testMessage);
        this.updateNotificationCount();
        this.addNotificationBadge();
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.notificationHandler = new NotificationHandler();
});

// Export for potential external use
if (typeof module !== 'undefined' && module.exports) {
    module.exports = NotificationHandler;
}



function _updateStatCounts(delta) {
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const weekStart = new Date(today.getTime() - (today.getDay() * 24 * 60 * 60 * 1000));
    const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);

    const totalEl = document.getElementById('totalCount');
    const todayEl = document.getElementById('todayCount');
    const weekEl = document.getElementById('weekCount');
    const monthEl = document.getElementById('monthCount');

    if (totalEl) totalEl.textContent = Math.max(0, (parseInt(totalEl.textContent) || 0) + delta);
    if (todayEl) todayEl.textContent = Math.max(0, (parseInt(todayEl.textContent) || 0) + delta);
    if (weekEl) weekEl.textContent = Math.max(0, (parseInt(weekEl.textContent) || 0) + delta);
    if (monthEl) monthEl.textContent = Math.max(0, (parseInt(monthEl.textContent) || 0) + delta);
}
