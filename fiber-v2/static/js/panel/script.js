/**
 * Hospital CMS Admin Panel JavaScript
 * Handles sidebar navigation, notifications, and interactive elements
 */

class AdminPanel {
    constructor() {
        this.sidebar = document.getElementById('adminSidebar');
        this.sidebarToggle = document.getElementById('sidebarToggle');
        this.notificationBtn = document.getElementById('notificationBtn');
        this.notificationDropdown = document.getElementById('notificationDropdown');
        this.header = document.querySelector('.admin-header');
        this.content = document.querySelector('.admin-content');
        
        this.init();
    }
    
    init() {
        this.setupSidebarToggle();
        this.setupNavigationGroups();
        this.setupNotifications();
        this.setupHeaderActions();
        this.setupResponsive();
        this.setupTooltips();
        this.loadSavedState();
        
        // Add smooth animations
        document.body.classList.add('loaded');
    }
    
    /**
     * Setup sidebar toggle functionality
     */
    setupSidebarToggle() {
        if (this.sidebarToggle) {
            this.sidebarToggle.addEventListener('click', () => {
                this.toggleSidebar();
            });
        }
        
        // Double-click to toggle
        if (this.sidebar) {
            this.sidebar.addEventListener('dblclick', (e) => {
                if (e.target.closest('.nav-group-header') || e.target.closest('.nav-single')) {
                    return;
                }
                this.toggleSidebar();
            });
        }
    }
    
    /**
     * Toggle sidebar collapsed state
     */
    toggleSidebar() {
        const isCollapsed = this.sidebar.classList.contains('collapsed');
        
        if (isCollapsed) {
            this.expandSidebar();
        } else {
            this.collapseSidebar();
        }
        
        // Save state
        localStorage.setItem('sidebarCollapsed', !isCollapsed);
    }
    
    /**
     * Collapse sidebar
     */
    collapseSidebar() {
        this.sidebar.classList.add('collapsed');
        this.header?.classList.add('sidebar-collapsed');
        this.content?.classList.add('sidebar-collapsed');
        
        // Update toggle icon
        const icon = this.sidebarToggle?.querySelector('i');
        if (icon) {
            icon.className = 'fas fa-chevron-right';
        }
        
        // Close all expanded groups
        this.closeAllNavGroups();
        
        // Add animation class
        this.sidebar.classList.add('collapsing');
        setTimeout(() => {
            this.sidebar.classList.remove('collapsing');
        }, 300);
    }
    
    /**
     * Expand sidebar
     */
    expandSidebar() {
        this.sidebar.classList.remove('collapsed');
        this.header?.classList.remove('sidebar-collapsed');
        this.content?.classList.remove('sidebar-collapsed');
        
        // Update toggle icon
        const icon = this.sidebarToggle?.querySelector('i');
        if (icon) {
            icon.className = 'fas fa-chevron-left';
        }
        
        // Add animation class
        this.sidebar.classList.add('expanding');
        setTimeout(() => {
            this.sidebar.classList.remove('expanding');
        }, 300);
    }
    
    /**
     * Setup navigation group functionality
     */
    setupNavigationGroups() {
        const navGroupHeaders = document.querySelectorAll('.nav-group-header');
        
        navGroupHeaders.forEach(header => {
            header.addEventListener('click', (e) => {
                e.preventDefault();
                const groupName = header.getAttribute('data-group');
                
                if (this.sidebar.classList.contains('collapsed')) {
                    return;
                }
                
                this.toggleNavGroup(groupName, header);
            });
        });
        
        // Setup nav item clicks
        const navItems = document.querySelectorAll('.nav-item, .nav-single');
        navItems.forEach(item => {
            item.addEventListener('click', (e) => {
                this.setActiveNavItem(item);
            });
        });
    }
    
    /**
     * Toggle navigation group
     */
    toggleNavGroup(groupName, header) {
        const groupItems = document.querySelector(`[data-group-items="${groupName}"]`);
        const isExpanded = header.classList.contains('expanded');

        if (!groupItems) return;

        const animateHeight = (el, toOpen) => {
            const startHeight = el.getBoundingClientRect().height;
            el.style.height = 'auto';
            const targetHeight = el.scrollHeight;
            el.style.height = `${startHeight}px`;
            el.offsetHeight; // force reflow
            el.classList.add(toOpen ? 'expanding' : 'collapsing');
            el.style.height = toOpen ? `${targetHeight}px` : '0px';
            const duration = 300;
            setTimeout(() => {
                el.classList.remove('expanding', 'collapsing');
                if (toOpen) {
                    el.classList.add('expanded');
                    el.style.height = 'auto';
                } else {
                    el.classList.remove('expanded');
                }
            }, duration);
        };

        if (isExpanded) {
            header.classList.remove('expanded');
            header.classList.remove('active');
            animateHeight(groupItems, false);
        } else {
            header.classList.add('expanded');
            header.classList.add('active');
            animateHeight(groupItems, true);
        }
        
        // Save expanded state
        this.saveNavGroupState(groupName, !isExpanded);
    }
    
    /**
     * Close all navigation groups
     */
    closeAllNavGroups() {
        const headers = document.querySelectorAll('.nav-group-header');
        const items = document.querySelectorAll('.nav-group-items');
        
        headers.forEach(header => {
            header.classList.remove('expanded', 'active');
        });
        
        items.forEach(item => {
            item.classList.remove('expanded');
        });
    }
    
    /**
     * Set active navigation item
     */
    setActiveNavItem(clickedItem) {
        // Remove active class from all items
        const allNavItems = document.querySelectorAll('.nav-item, .nav-single');
        allNavItems.forEach(item => {
            item.classList.remove('active');
        });
        
        // Add active class to clicked item
        clickedItem.classList.add('active');
        
        // If it's a nav-item, also activate its parent group
        if (clickedItem.classList.contains('nav-item')) {
            const parentGroup = clickedItem.closest('.nav-group');
            if (parentGroup) {
                const groupHeader = parentGroup.querySelector('.nav-group-header');
                if (groupHeader) {
                    groupHeader.classList.add('active');
                }
            }
        }
    }
    
    /**
     * Setup notification functionality
     */
    setupNotifications() {
        if (this.notificationBtn && this.notificationDropdown) {
            this.notificationBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                this.toggleNotifications();
            });
            
            // Close notifications when clicking outside
            document.addEventListener('click', (e) => {
                if (!this.notificationDropdown.contains(e.target) && 
                    !this.notificationBtn.contains(e.target)) {
                    this.closeNotifications();
                }
            });
            
            // Handle notification item clicks
            const notificationItems = this.notificationDropdown.querySelectorAll('.notification-item');
            notificationItems.forEach(item => {
                item.addEventListener('click', () => {
                    this.markNotificationAsRead(item);
                });
            });
        }
    }
    
    /**
     * Toggle notification dropdown
     */
    toggleNotifications() {
        const isVisible = this.notificationDropdown.classList.contains('show');
        
        if (isVisible) {
            this.closeNotifications();
        } else {
            this.openNotifications();
        }
    }
    
    /**
     * Open notification dropdown
     */
    openNotifications() {
        this.notificationDropdown.classList.add('show');
        this.notificationDropdown.classList.add('slide-down');
        
        // Update notification count
        this.updateNotificationCount();
    }
    
    /**
     * Close notification dropdown
     */
    closeNotifications() {
        this.notificationDropdown.classList.remove('show');
        setTimeout(() => {
            this.notificationDropdown.classList.remove('slide-down');
        }, 300);
    }
    
    /**
     * Mark notification as read
     */
    markNotificationAsRead(notificationItem) {
        if (notificationItem.classList.contains('unread')) {
            notificationItem.classList.remove('unread');
            this.updateNotificationCount();
            
            // Add read animation
            notificationItem.style.background = '#f0f9ff';
            setTimeout(() => {
                notificationItem.style.background = '';
            }, 1000);
        }
    }
    
    /**
     * Update notification count
     */
    updateNotificationCount() {
        const unreadItems = this.notificationDropdown.querySelectorAll('.notification-item.unread');
        const countElement = this.notificationDropdown.querySelector('.notification-count');
        
        if (countElement) {
            countElement.textContent = unreadItems.length;
        }
        
        // Only remove the notification indicator if there are no unread items
        // Don't add it here - it should only be added by backend or websocket
        if (unreadItems.length === 0) {
            this.notificationBtn.classList.remove('has-notification');
        }
    }
    
    /**
     * Setup header action buttons
     */
    setupHeaderActions() {
        const searchBtn = document.getElementById('searchBtn');
        const settingsBtn = document.getElementById('settingsBtn');
        const userProfile = document.getElementById('userProfile');
        
        if (searchBtn) {
            searchBtn.addEventListener('click', () => {
                this.handleSearch();
            });
        }
        
        if (settingsBtn) {
            settingsBtn.addEventListener('click', () => {
                this.handleSettings();
            });
        }
        
        if (userProfile) {
            userProfile.addEventListener('click', () => {
                this.handleUserProfile();
            });
        }
    }
    
    /**
     * Handle search functionality
     */
    handleSearch() {
        // Create search modal or redirect to search page
        console.log('Search functionality');
        // You can implement a search modal here
    }
    
    /**
     * Handle settings
     */
    handleSettings() {
        window.location.href = '/panel/settings';
    }
    
    /**
     * Handle user profile
     */
    handleUserProfile() {
        // Create user profile dropdown or redirect
        console.log('User profile clicked');
        // You can implement a user profile dropdown here
    }
    
    /**
     * Setup responsive behavior
     */
    setupResponsive() {
        const mediaQuery = window.matchMedia('(max-width: 768px)');
        
        const handleResponsive = (e) => {
            if (e.matches) {
                // Mobile view
                this.sidebar.classList.add('mobile');
                this.sidebar.classList.remove('collapsed');
            } else {
                // Desktop view
                this.sidebar.classList.remove('mobile');
            }
        };
        
        mediaQuery.addListener(handleResponsive);
        handleResponsive(mediaQuery);
        
        // Mobile menu toggle
        if (window.innerWidth <= 768) {
            this.setupMobileMenu();
        }
    }
    
    /**
     * Setup mobile menu
     */
    setupMobileMenu() {
        // Create mobile menu button if it doesn't exist
        let mobileMenuBtn = document.querySelector('.mobile-menu-btn');
        
        if (!mobileMenuBtn) {
            mobileMenuBtn = document.createElement('button');
            mobileMenuBtn.className = 'header-btn mobile-menu-btn';
            mobileMenuBtn.innerHTML = '<i class="fas fa-bars"></i>';
            mobileMenuBtn.title = 'Menü';
            
            const headerLeft = document.querySelector('.header-left');
            if (headerLeft) {
                headerLeft.insertBefore(mobileMenuBtn, headerLeft.firstChild);
            }
        }
        
        mobileMenuBtn.addEventListener('click', () => {
            this.sidebar.classList.toggle('mobile-open');
        });
        
        // Close mobile menu when clicking outside
        document.addEventListener('click', (e) => {
            if (window.innerWidth <= 768) {
                if (!this.sidebar.contains(e.target) && !mobileMenuBtn.contains(e.target)) {
                    this.sidebar.classList.remove('mobile-open');
                }
            }
        });
    }
    
    /**
     * Setup tooltips for collapsed sidebar
     */
    setupTooltips() {
        // Tooltips are handled via CSS, but we can add dynamic behavior here if needed
    }
    
    /**
     * Save navigation group state
     */
    saveNavGroupState(groupName, isExpanded) {
        const expandedGroups = JSON.parse(localStorage.getItem('expandedNavGroups') || '[]');
        
        if (isExpanded && !expandedGroups.includes(groupName)) {
            expandedGroups.push(groupName);
        } else if (!isExpanded) {
            const index = expandedGroups.indexOf(groupName);
            if (index > -1) {
                expandedGroups.splice(index, 1);
            }
        }
        
        localStorage.setItem('expandedNavGroups', JSON.stringify(expandedGroups));
    }
    
    /**
     * Load saved state from localStorage
     */
    loadSavedState() {
        // Load sidebar collapsed state
        const sidebarCollapsed = localStorage.getItem('sidebarCollapsed') === 'true';
        if (sidebarCollapsed) {
            this.collapseSidebar();
        }
        
        // Load expanded nav groups (do not mark as active here; just expand)
        const expandedGroups = JSON.parse(localStorage.getItem('expandedNavGroups') || '[]');
        expandedGroups.forEach(groupName => {
            const header = document.querySelector(`[data-group="${groupName}"]`);
            const items = document.querySelector(`[data-group-items="${groupName}"]`);
            
            if (header && items && !this.sidebar.classList.contains('collapsed')) {
                header.classList.add('expanded');
                items.classList.add('expanded');
            }
        });
        
        // Set active nav item based on current URL
        this.setActiveNavFromURL();
    }
    
    /**
     * Set active navigation item based on current URL
     */
    setActiveNavFromURL() {
        const currentPath = window.location.pathname;
        const navItems = Array.from(document.querySelectorAll('.nav-item, .nav-single'));

        // Clear any existing active classes first
        navItems.forEach(item => item.classList.remove('active'));
        document.querySelectorAll('.nav-group-header').forEach(h => h.classList.remove('active'));

        // Find the best match (longest href that matches the current path)
        let bestItem = null;
        let bestScore = -1;

        navItems.forEach(item => {
            const href = item.getAttribute('href') || '';
            if (!href) return;
            const exact = currentPath === href;
            const starts = currentPath.startsWith(href + '/');
            if (exact || starts) {
                const score = href.length; // prefer deeper paths
                if (score > bestScore) {
                    bestScore = score;
                    bestItem = item;
                }
            }
        });

        if (bestItem) {
            this.setActiveNavItem(bestItem);
        }
    }
    
    /**
     * Add notification programmatically
     */
    addNotification(type, message, time = 'Şimdi') {
        const notificationList = this.notificationDropdown?.querySelector('.notification-list');
        if (!notificationList) return;
        
        const iconMap = {
            success: 'fas fa-check-circle',
            warning: 'fas fa-exclamation-triangle',
            danger: 'fas fa-times-circle',
            info: 'fas fa-info-circle'
        };
        
        const notificationItem = document.createElement('div');
        notificationItem.className = 'notification-item unread';
        notificationItem.innerHTML = `
            <div class="notification-icon ${type}">
                <i class="${iconMap[type] || iconMap.info}"></i>
            </div>
            <div class="notification-content">
                <div class="notification-text">${message}</div>
                <div class="notification-time">${time}</div>
            </div>
        `;
        
        // Add click handler
        notificationItem.addEventListener('click', () => {
            this.markNotificationAsRead(notificationItem);
        });
        
        // Insert at the beginning
        notificationList.insertBefore(notificationItem, notificationList.firstChild);
        
        // Update count
        this.updateNotificationCount();
        
        // Show animation
        notificationItem.classList.add('fade-in');
    }
    
    /**
     * Show loading state
     */
    showLoading(element) {
        if (element) {
            element.classList.add('loading');
        }
    }
    
    /**
     * Hide loading state
     */
    hideLoading(element) {
        if (element) {
            element.classList.remove('loading');
        }
    }
}

// Initialize admin panel when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.adminPanel = new AdminPanel();
});

// Utility functions
window.AdminPanelUtils = {
    /**
     * Show toast notification
     */
    showToast(message, type = 'info', duration = 3000) {
        // You can implement a toast notification system here
        console.log(`${type.toUpperCase()}: ${message}`);
    },
    
    /**
     * Confirm dialog
     */
    confirm(message, callback) {
        if (confirm(message)) {
            callback();
        }
    },
    
    /**
     * Format date
     */
    formatDate(date) {
        return new Intl.DateTimeFormat('tr-TR', {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        }).format(new Date(date));
    }
};
  