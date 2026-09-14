/**
 * Contact Request (Individual Contact Request View) Page JavaScript
 * Handles delete functionality, modal interactions, and AJAX operations
 */

class ContactRequestViewManager {
    constructor() {
        this.contactRequestData = window.contactRequestData || {};
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupDeleteButton();
        this.setupDropdown();
        this.setupToggleReadStatus();
        this.setupKeyboardShortcuts();
    }

    /**
     * Setup all event listeners
     */
    setupEventListeners() {
        // Handle escape key for modals
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                const deleteModal = document.getElementById('deleteModal');

                if (deleteModal && deleteModal.style.display !== 'none') {
                    this.closeDeleteModal();
                } else {
                    this.closeDropdown();
                }
            }
        });

        // Close modals when clicking outside
        document.addEventListener('click', (e) => {
            const deleteModal = document.getElementById('deleteModal');

            if (e.target === deleteModal) {
                this.closeDeleteModal();
            }
        });

        // Handle window resize
        window.addEventListener('resize', () => {
            this.handleResize();
        });
    }

    /**
     * Setup delete button functionality
     */
    setupDeleteButton() {
        const deleteBtn = document.getElementById('deleteContactRequestBtn');
        if (!deleteBtn) return;
        
        deleteBtn.addEventListener('click', (e) => {
            e.preventDefault();
            this.showDeleteModal();
        });
        
        // Setup confirm delete button
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        if (confirmBtn) {
            confirmBtn.addEventListener('click', (e) => {
                e.preventDefault();
                this.deleteContactRequest();
            });
        }
    }

    /**
     * Setup dropdown functionality
     */
    setupDropdown() {
        const dropdownToggle = document.getElementById('contactRequestOptionsDropdown');
        const dropdownMenu = document.querySelector('#contactRequestOptionsDropdown + .dropdown-menu');
        
        if (!dropdownToggle || !dropdownMenu) return;
        
        dropdownToggle.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            this.toggleDropdown();
        });
        
        // Close dropdown when clicking outside
        document.addEventListener('click', (e) => {
            if (!dropdownToggle.contains(e.target) && !dropdownMenu.contains(e.target)) {
                this.closeDropdown();
            }
        });
    }

    /**
     * Toggle dropdown menu
     */
    toggleDropdown() {
        const dropdownMenu = document.querySelector('#contactRequestOptionsDropdown + .dropdown-menu');
        const dropdownToggle = document.getElementById('contactRequestOptionsDropdown');
        
        if (!dropdownMenu || !dropdownToggle) return;
        
        const isOpen = dropdownMenu.classList.contains('show');
        
        if (isOpen) {
            this.closeDropdown();
        } else {
            this.openDropdown();
        }
    }

    /**
     * Open dropdown menu
     */
    openDropdown() {
        const dropdownMenu = document.querySelector('#contactRequestOptionsDropdown + .dropdown-menu');
        const dropdownToggle = document.getElementById('contactRequestOptionsDropdown');
        
        if (!dropdownMenu || !dropdownToggle) return;
        
        dropdownMenu.classList.add('show');
        dropdownToggle.setAttribute('aria-expanded', 'true');
    }

    /**
     * Close dropdown menu
     */
    closeDropdown() {
        const dropdownMenu = document.querySelector('#contactRequestOptionsDropdown + .dropdown-menu');
        const dropdownToggle = document.getElementById('contactRequestOptionsDropdown');
        
        if (!dropdownMenu || !dropdownToggle) return;
        
        dropdownMenu.classList.remove('show');
        dropdownToggle.setAttribute('aria-expanded', 'false');
    }

    /**
     * Setup toggle read status functionality
     */
    setupToggleReadStatus() {
        const toggleBtn = document.getElementById('toggleReadStatusBtn');
        if (!toggleBtn) return;
        
        toggleBtn.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            this.toggleReadStatus();
        });
    }

    /**
     * Show delete confirmation modal
     */
    showDeleteModal() {
        const modal = document.getElementById('deleteModal');
        const nameSpan = document.getElementById('deleteItemName');
        
        if (!modal) return;
        
        if (nameSpan) {
            nameSpan.textContent = this.contactRequestData.name || 'Bu iletişim talebi';
        }
        
        modal.style.display = 'flex';
        document.body.style.overflow = 'hidden';
    }

    /**
     * Close delete modal
     */
    closeDeleteModal() {
        const modal = document.getElementById('deleteModal');
        if (!modal) return;
        
        modal.style.display = 'none';
        document.body.style.overflow = '';
    }

    /**
     * Toggle read status
     */
    async toggleReadStatus() {
        const toggleBtn = document.getElementById('toggleReadStatusBtn');
        if (!toggleBtn) return;
        
        const crid = toggleBtn.getAttribute('data-id');
        const isRead = toggleBtn.getAttribute('data-is-read') === 'true';
        const targetIsRead = !isRead; // send desired state to backend
        
        this.setButtonLoading(toggleBtn, true);
        
        try {
            const response = await fetch(`/backend/contact-request/${crid}/set-as-read`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                },
                body: JSON.stringify({ is_read: targetIsRead })
            });

            const responseData = await response.json().catch(() => ({}));

            if (responseData.status === 201) {
                this.showAlert(responseData.message, 'success');
                // Reload page to reflect changes
                setTimeout(() => {
                    window.location.reload();
                }, 1500);
            } else {
                this.showAlert(responseData.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            }
        } catch (error) {
            console.error('Toggle read status error:', error);
            this.showAlert('Bağlantı hatası oluştu.', 'error');
        } finally {
            this.setButtonLoading(toggleBtn, false);
        }
    }

    /**
     * Delete contact request
     */
    async deleteContactRequest() {
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        if (!confirmBtn) return;
        
        this.setButtonLoading(confirmBtn, true);
        
        try {
            const response = await fetch(`/backend/contact-request/${this.contactRequestData.id}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });
            
            const data = await response.json();
            
            if (data.status === 201) {
                this.showAlert('İletişim talebi başarıyla silindi.', 'success');
                setTimeout(() => {
                    window.location.href = '/panel/iletisim-istekleri';
                }, 1500);
            } else {
                this.showAlert(data.message || 'İletişim talebi silinirken hata oluştu.', 'error');
                this.setButtonLoading(confirmBtn, false);
            }
        } catch (error) {
            console.error('Error deleting contact request:', error);
            this.showAlert('Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            this.setButtonLoading(confirmBtn, false);
        }
    }

    /**
     * Set button loading state
     */
    setButtonLoading(button, loading) {
        if (!button) return;
        
        const textSpan = button.querySelector('.btn-text');
        const loader = button.querySelector('.btn-loader');
        
        if (loading) {
            button.disabled = true;
            if (textSpan) textSpan.style.display = 'none';
            if (loader) loader.style.display = 'inline-flex';
        } else {
            button.disabled = false;
            if (textSpan) textSpan.style.display = 'inline';
            if (loader) loader.style.display = 'none';
        }
    }

    /**
     * Show alert message
     */
    showAlert(message, type = 'success') {
        const alertId = type === 'success' ? 'successMessage' : 'errorMessage';
        const alert = document.getElementById(alertId);
        
        if (!alert) return;
        
        const textSpan = alert.querySelector('.alert-text');
        if (textSpan) {
            textSpan.textContent = message;
        }
        
        alert.style.display = 'flex';
        
        // Auto hide after 5 seconds
        setTimeout(() => {
            this.closeAlert(alertId);
        }, 5000);
    }

    /**
     * Close alert message
     */
    closeAlert(alertId) {
        const alert = document.getElementById(alertId);
        if (alert) {
            alert.style.display = 'none';
        }
    }

    /**
     * Setup keyboard shortcuts
     */
    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            // Ctrl/Cmd + D for delete
            if ((e.ctrlKey || e.metaKey) && e.key === 'd') {
                e.preventDefault();
                this.showDeleteModal();
            }
            
            // Ctrl/Cmd + R for toggle read status
            if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
                e.preventDefault();
                this.toggleReadStatus();
            }
        });
    }

    /**
     * Handle window resize
     */
    handleResize() {
        // Adjust modal positions if needed
        const modals = document.querySelectorAll('.modal');
        modals.forEach(modal => {
            if (modal.style.display !== 'none') {
                // Recalculate modal position if needed
            }
        });
    }
}

// Global functions for HTML onclick attributes
function closeDeleteModal() {
    if (window.contactRequestViewManager) {
        window.contactRequestViewManager.closeDeleteModal();
    }
}

function closeAlert(alertId) {
    if (window.contactRequestViewManager) {
        window.contactRequestViewManager.closeAlert(alertId);
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.contactRequestViewManager = new ContactRequestViewManager();
});

