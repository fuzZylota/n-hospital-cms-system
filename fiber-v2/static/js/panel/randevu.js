/**
 * Randevu (Individual Appointment View) Page JavaScript
 * Handles delete functionality, modal interactions, and AJAX operations
 */

class RandevuViewManager {
    constructor() {
        this.randevuData = window.randevuData || {};
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupDeleteButton();
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
        const deleteBtn = document.getElementById('deleteRandevuBtn');
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
                this.deleteRandevu();
            });
        }
    }

    /**
     * Show delete confirmation modal
     */
    showDeleteModal() {
        const modal = document.getElementById('deleteModal');
        const nameSpan = document.getElementById('deleteItemName');
        
        if (!modal) return;
        
        if (nameSpan) {
            nameSpan.textContent = this.randevuData.name || 'Bu randevu';
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
     * Delete randevu
     */
    async deleteRandevu() {
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        if (!confirmBtn) return;
        
        this.setButtonLoading(confirmBtn, true);
        
        try {
            const response = await fetch(`/backend/randevu/${this.randevuData.id}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
            });
            
            const data = await response.json();
            
            if (data.status === 200) {
                this.showAlert('Randevu başarıyla silindi.', 'success');
                setTimeout(() => {
                    window.location.href = '/panel/randevular';
                }, 1500);
            } else {
                this.showAlert(data.message || 'Randevu silinirken hata oluştu.', 'error');
                this.setButtonLoading(confirmBtn, false);
            }
        } catch (error) {
            console.error('Error deleting randevu:', error);
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
        alert.classList.add('show');
        
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
            alert.classList.remove('show');
            setTimeout(() => {
                alert.style.display = 'none';
            }, 300);
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
            
            // Ctrl/Cmd + E for edit
            if ((e.ctrlKey || e.metaKey) && e.key === 'e') {
                e.preventDefault();
                const editLink = document.querySelector('a[href*="/duzenle"]');
                if (editLink) {
                    editLink.click();
                }
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
    if (window.randevuViewManager) {
        window.randevuViewManager.closeDeleteModal();
    }
}

function closeAlert(alertId) {
    if (window.randevuViewManager) {
        window.randevuViewManager.closeAlert(alertId);
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.randevuViewManager = new RandevuViewManager();
});
