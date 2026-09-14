/**
 * Panel Dashboard Manager
 * Handles dashboard statistics display and refresh functionality
 */

class PanelDashboardManager {
    constructor() {
        this.refreshBtn = document.getElementById('refreshStats');
        this.dashboard = document.querySelector('.panel-dashboard');
        this.init();
    }

    init() {
        this.bindEvents();
        this.animateStats();
        this.setupTooltips();
    }

    bindEvents() {
        if (this.refreshBtn) {
            this.refreshBtn.addEventListener('click', () => this.refreshStats());
        }

        // Add keyboard shortcut for refresh (Ctrl/Cmd + R)
        document.addEventListener('keydown', (e) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
                e.preventDefault();
                this.refreshStats();
            }
        });
    }

    /**
     * Refresh statistics by reloading the page
     */
    async refreshStats() {
        if (!this.refreshBtn) return;

        // Add loading state
        this.refreshBtn.disabled = true;
        const icon = this.refreshBtn.querySelector('i');
        const originalClass = icon.className;
        icon.className = 'fas fa-spinner fa-spin';

        try {
            // Reload the page to get fresh statistics
            window.location.reload();
        } catch (error) {
            console.error('Error refreshing statistics:', error);
            this.showNotification('İstatistikler yüklenirken bir hata oluştu', 'error');
            
            // Restore button state
            this.refreshBtn.disabled = false;
            icon.className = originalClass;
        }
    }

    /**
     * Animate statistics on page load
     */
    animateStats() {
        // Animate stat values with counting effect
        const statValues = document.querySelectorAll('.stat-value, .mini-value');
        
        statValues.forEach((element, index) => {
            const text = element.textContent.trim();
            const number = parseFloat(text.replace(/[^0-9.-]/g, ''));
            
            // Only animate if it's a valid number and not too large
            if (!isNaN(number) && number < 10000 && number >= 0) {
                element.textContent = '0';
                
                setTimeout(() => {
                    this.animateValue(element, 0, number, text, 800);
                }, index * 20); // Stagger animations
            }
        });

        // Fade in cards
        const cards = document.querySelectorAll('.stat-card, .stats-section');
        cards.forEach((card, index) => {
            card.style.opacity = '0';
            card.style.transform = 'translateY(20px)';
            
            setTimeout(() => {
                card.style.transition = 'opacity 0.5s ease, transform 0.5s ease';
                card.style.opacity = '1';
                card.style.transform = 'translateY(0)';
            }, index * 30);
        });
    }

    /**
     * Animate a number from start to end
     */
    animateValue(element, start, end, originalText, duration) {
        const startTime = performance.now();
        const suffix = originalText.replace(/[0-9.,\s-]/g, '');
        
        const animate = (currentTime) => {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            
            // Easing function (easeOutQuad)
            const easeProgress = progress * (2 - progress);
            const current = start + (end - start) * easeProgress;
            
            // Format number with appropriate decimal places
            let formattedValue;
            if (originalText.includes(',')) {
                formattedValue = current.toLocaleString('tr-TR', {
                    minimumFractionDigits: 0,
                    maximumFractionDigits: 2
                });
            } else {
                formattedValue = Math.round(current).toString();
            }
            
            element.textContent = formattedValue + suffix;
            
            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };
        
        requestAnimationFrame(animate);
    }

    /**
     * Setup tooltips for statistics
     */
    setupTooltips() {
        const miniStats = document.querySelectorAll('.mini-stat');
        
        miniStats.forEach(stat => {
            const label = stat.querySelector('.mini-label');
            const value = stat.querySelector('.mini-value');
            
            if (label && value) {
                stat.setAttribute('title', `${label.textContent}: ${value.textContent}`);
            }
        });
    }

    /**
     * Show notification
     */
    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.innerHTML = `
            <i class="fas fa-${this.getIconForType(type)}"></i>
            <span>${message}</span>
        `;
        
        // Add to page
        document.body.appendChild(notification);
        
        // Animate in
        setTimeout(() => {
            notification.classList.add('show');
        }, 10);
        
        // Remove after delay
        setTimeout(() => {
            notification.classList.remove('show');
            setTimeout(() => {
                notification.remove();
            }, 300);
        }, 3000);
    }

    /**
     * Get icon class for notification type
     */
    getIconForType(type) {
        const icons = {
            success: 'check-circle',
            error: 'exclamation-circle',
            warning: 'exclamation-triangle',
            info: 'info-circle'
        };
        return icons[type] || 'info-circle';
    }

    /**
     * Export statistics to CSV
     */
    exportToCSV() {
        const rows = [];
        const sections = document.querySelectorAll('.stats-section');
        
        sections.forEach(section => {
            const title = section.querySelector('.section-header h2').textContent.trim();
            rows.push([title, '', '']);
            
            const stats = section.querySelectorAll('.mini-stat');
            stats.forEach(stat => {
                const label = stat.querySelector('.mini-label').textContent.trim();
                const value = stat.querySelector('.mini-value').textContent.trim();
                rows.push(['', label, value]);
            });
            
            rows.push(['', '', '']); // Empty row between sections
        });
        
        // Convert to CSV
        const csv = rows.map(row => row.join(',')).join('\n');
        
        // Download
        const blob = new Blob(['\ufeff' + csv], { type: 'text/csv;charset=utf-8;' });
        const link = document.createElement('a');
        link.href = URL.createObjectURL(blob);
        link.download = `panel-istatistikleri-${new Date().toISOString().split('T')[0]}.csv`;
        link.click();
    }

    /**
     * Print dashboard
     */
    printDashboard() {
        window.print();
    }
}

// Initialize dashboard manager when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.panelDashboard = new PanelDashboardManager();
});

// Add notification styles dynamically
const notificationStyles = `
    .notification {
        position: fixed;
        top: 20px;
        right: 20px;
        background: white;
        padding: 16px 20px;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        display: flex;
        align-items: center;
        gap: 12px;
        z-index: 10000;
        opacity: 0;
        transform: translateX(400px);
        transition: all 0.3s ease;
    }
    
    .notification.show {
        opacity: 1;
        transform: translateX(0);
    }
    
    .notification i {
        font-size: 20px;
    }
    
    .notification-success {
        border-left: 4px solid var(--success-color);
    }
    
    .notification-success i {
        color: var(--success-color);
    }
    
    .notification-error {
        border-left: 4px solid var(--danger-color);
    }
    
    .notification-error i {
        color: var(--danger-color);
    }
    
    .notification-warning {
        border-left: 4px solid var(--warning-color);
    }
    
    .notification-warning i {
        color: var(--warning-color);
    }
    
    .notification-info {
        border-left: 4px solid var(--info-color);
    }
    
    .notification-info i {
        color: var(--info-color);
    }
`;

// Inject notification styles
const styleElement = document.createElement('style');
styleElement.textContent = notificationStyles;
document.head.appendChild(styleElement);






