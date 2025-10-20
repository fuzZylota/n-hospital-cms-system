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
            
            if (message.message) {
                this.addNotificationToDropdown(message);
                this.updateNotificationCount();
                this.addNotificationBadge();
            }
        } catch (error) {
            console.error('[Notifications] Error parsing message:', error);
        }
    }

    addNotificationToDropdown(message) {
        if (!this.notificationList) return;

        // Create notification item
        const notificationItem = document.createElement('div');
        notificationItem.className = 'notification-item unread';
        
        const iconClass = this.getIconClass(message.notification_type || 'info');
        const timeText = this.formatTime(new Date());
        
        notificationItem.innerHTML = `
            <div class="notification-icon ${message.notification_type || 'info'}">
                <i class="fas ${iconClass}"></i>
            </div>
            <div class="notification-content">
                <div class="notification-text">${message.message}</div>
                <div class="notification-time">${timeText}</div>
            </div>
        `;

        // Add click handler for notification link
        if (message.request_link) {
            notificationItem.style.cursor = 'pointer';
            notificationItem.addEventListener('click', () => {
                window.location.href = message.request_link;
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


