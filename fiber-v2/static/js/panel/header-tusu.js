/**
 * Header Tusu (Individual Header Button View) Page JavaScript
 * Handles delete functionality, sort order change, copy to clipboard, and interactive elements
 */

class HeaderButtonViewManager {
    constructor() {
        this.headerButtonData = window.headerButtonData || {};
        this.orderModal = document.getElementById('orderChangeModal');
        this.orderForm = document.getElementById('orderChangeForm');
        this.newSortOrderInput = document.getElementById('new_sort_order');
        this.parentIdSelect = document.getElementById('parent_id');
        this.oldSortOrderInput = document.getElementById('old_sort_order');
        this.oldParentIdInput = document.getElementById('old_parent_id');
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupDeleteButton();
        this.setupOrderChangeButtons();
        this.setupCopyToClipboard();
        this.loadParentOptions();
    }

    /**
     * Setup all event listeners
     */
    setupEventListeners() {
        // Handle escape key for modals
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                const deleteModal = document.getElementById('deleteModal');
                const orderModal = document.getElementById('orderChangeModal');
                
                if (deleteModal.classList.contains('show')) {
                    this.closeDeleteModal();
                } else if (orderModal.classList.contains('show')) {
                    this.closeOrderModal();
                }
            }
        });

        // Close modals when clicking outside
        document.addEventListener('click', (e) => {
            const deleteModal = document.getElementById('deleteModal');
            const orderModal = document.getElementById('orderChangeModal');
            
            if (e.target === deleteModal) {
                this.closeDeleteModal();
            } else if (e.target === orderModal) {
                this.closeOrderModal();
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
        const deleteBtn = document.getElementById('deleteHeaderButtonBtn') || document.getElementById('deleteIndividualHeaderButtonBtn');
        
        if (!deleteBtn) return;

        deleteBtn.addEventListener('click', () => {
            const id = deleteBtn.getAttribute('data-id');
            const name = deleteBtn.getAttribute('data-name');
            this.showDeleteConfirmation(id, name);
        });
    }

    /**
     * Setup order change buttons
     */
    setupOrderChangeButtons() {
        const changeOrderBtn = document.getElementById('changeOrderBtn');
        const changeOrderBtnInline = document.getElementById('changeOrderBtnInline');
        const confirmOrderChangeBtn = document.getElementById('confirmOrderChangeBtn');

        if (changeOrderBtn) {
            changeOrderBtn.addEventListener('click', () => {
                this.showOrderChangeModal();
            });
        }

        if (changeOrderBtnInline) {
            changeOrderBtnInline.addEventListener('click', () => {
                this.showOrderChangeModal();
            });
        }

        if (confirmOrderChangeBtn) {
            confirmOrderChangeBtn.addEventListener('click', () => {
                this.changeOrder();
            });
        }
    }

    /**
     * Load parent options for the select dropdown
     */
    async loadParentOptions() {
        try {
            const response = await fetch('/backend/header-buttons/parents', {
                method: 'GET',
                headers: {
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            if (response.ok) {
                const data = await response.json();
                if (data.status === 200 && data.data) {
                    this.populateParentOptions(data.data);
                }
            }
        } catch (error) {
            console.error('Failed to load parent options:', error);
        }
    }

    /**
     * Populate parent options in select dropdown
     */
    populateParentOptions(parents) {
        if (!this.parentIdSelect) return;

        // Clear existing options except the first one
        this.parentIdSelect.innerHTML = '<option value="">Ana Menü</option>';

        parents.forEach(parent => {
            const option = document.createElement('option');
            option.value = parent.hbid;
            option.textContent = parent.title;
            option.selected = parent.hbid === this.headerButtonData.parentId;
            this.parentIdSelect.appendChild(option);
        });
    }

    /**
     * Show order change modal
     */
    showOrderChangeModal() {
        this.orderModal.style.display = 'flex';
        setTimeout(() => this.orderModal.classList.add('show'), 10);
        
        // Reset form with current values
        this.newSortOrderInput.value = this.headerButtonData.currentSortOrder;
        this.parentIdSelect.value = this.headerButtonData.parentId || '';
        
        // Reset field states
        this.newSortOrderInput.classList.remove('error', 'success');
        this.parentIdSelect.classList.remove('error', 'success');
        
        // Focus first input
        setTimeout(() => this.newSortOrderInput.focus(), 100);
        
        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Close order change modal
     */
    closeOrderModal() {
        this.orderModal.classList.remove('show');
        setTimeout(() => {
            this.orderModal.style.display = 'none';
        }, 300);
        
        // Restore body scroll
        document.body.style.overflow = '';
    }

    /**
     * Change sort order
     */
    async changeOrder() {
        const newSortOrder = parseInt(this.newSortOrderInput.value);
        const parentId = this.parentIdSelect.value;
        const oldSortOrder = parseInt(this.oldSortOrderInput.value);
        const oldParentId = this.oldParentIdInput.value;

        // Validate inputs
        if (!newSortOrder || newSortOrder < 1) {
            this.showAlert('Sıra numarası 1\'den büyük olmalıdır.', 'error');
            this.setFieldState(this.newSortOrderInput, false);
            return;
        }

        const confirmBtn = document.getElementById('confirmOrderChangeBtn');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');

        try {
            // Show loading state
            confirmBtn.disabled = true;
            btnText.style.opacity = '0';
            btnLoader.style.display = 'block';

            const response = await fetch(`/backend/header-button/${this.headerButtonData.id}/change-order`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                },
                body: JSON.stringify({
                    parent_id: parentId || null,
                    new_sort_order: newSortOrder,
                    old_sort_order: oldSortOrder,
                    old_parent_id: oldParentId || null
                })
            });

            const result = await response.json().catch(() => ({}));

            if (result.status === 201) {
                this.closeOrderModal();
                this.showAlert('Sıra başarıyla değiştirildi.', 'success');
                
                // Update the current sort order in the data
                this.headerButtonData.currentSortOrder = newSortOrder;
                this.headerButtonData.parentId = parentId;
                
                // Update the display
                this.updateOrderDisplay();
                
                // Reload the page after 2 seconds to show updated data
                setTimeout(() => {
                    window.location.reload();
                }, 2000);
            } else {
                this.showAlert(result.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            }
        } catch (error) {
            console.error('Order change error:', error);
            this.showAlert('Bağlantı hatası oluştu.', 'error');
        } finally {
            // Reset button state
            confirmBtn.disabled = false;
            btnText.style.opacity = '1';
            btnLoader.style.display = 'none';
        }
    }

    /**
     * Update order display in the page
     */
    updateOrderDisplay() {
        // Update sort order badge
        const sortOrderBadge = document.querySelector('.sort-order-badge');
        if (sortOrderBadge) {
            sortOrderBadge.textContent = this.headerButtonData.currentSortOrder;
        }

        // Update current order display
        const currentOrder = document.querySelector('.current-order');
        if (currentOrder) {
            currentOrder.textContent = this.headerButtonData.currentSortOrder;
        }

        // Update parent display
        const parentBadge = document.querySelector('.parent-badge');
        if (parentBadge) {
            if (this.headerButtonData.parentId) {
                parentBadge.className = 'parent-badge';
                parentBadge.innerHTML = `
                    <i class="fas fa-level-up-alt"></i>
                    Alt Menü (ID: ${this.headerButtonData.parentId})
                `;
            } else {
                parentBadge.className = 'parent-badge main';
                parentBadge.innerHTML = `
                    <i class="fas fa-home"></i>
                    Ana Menü
                `;
            }
        }
    }

    /**
     * Set field validation state
     */
    setFieldState(field, isValid) {
        field.classList.remove('error', 'success');
        field.classList.add(isValid ? 'success' : 'error');
    }

    /**
     * Show delete confirmation modal
     */
    showDeleteConfirmation(id, name) {
        const modal = document.getElementById('deleteModal');
        const nameSpan = document.getElementById('deleteItemName');
        const confirmBtn = document.getElementById('confirmDeleteBtn');

        if (!modal || !nameSpan || !confirmBtn) return;

        nameSpan.textContent = name;
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);

        // Remove existing listeners
        const newConfirmBtn = confirmBtn.cloneNode(true);
        confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);

        // Add new listener
        newConfirmBtn.addEventListener('click', () => {
            this.deleteHeaderButton(id);
        });

        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Delete header button
     */
    async deleteHeaderButton(id) {
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');

        if (!confirmBtn || !btnText || !btnLoader) return;

        try {
            // Show loading state
            confirmBtn.disabled = true;
            btnText.style.opacity = '0';
            btnLoader.style.display = 'block';

            const response = await fetch(`/backend/header-button/${id}/delete`, {
                method: 'POST',
                headers: {
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const responseData = await response.json().catch(() => ({}));
            
            if (responseData.status === 201) {
                this.closeDeleteModal();
                this.showAlert('Header button başarıyla silindi.', 'success');
                
                // Redirect to header buttons list after 2 seconds
                setTimeout(() => {
                    window.location.href = '/panel/header-tuslari';
                }, 2000);

            } else {
                this.showAlert(responseData.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            }
        } catch (error) {
            console.error('Delete error:', error);
            this.showAlert('Bağlantı hatası oluştu.', 'error');
        } finally {
            // Reset button state
            confirmBtn.disabled = false;
            btnText.style.opacity = '1';
            btnLoader.style.display = 'none';
        }
    }

    /**
     * Close delete modal
     */
    closeDeleteModal() {
        const modal = document.getElementById('deleteModal');
        if (!modal) return;

        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
        
        // Restore body scroll
        document.body.style.overflow = '';
    }

    /**
     * Setup copy to clipboard functionality
     */
    setupCopyToClipboard() {
        const copyButtons = document.querySelectorAll('.copy-btn');
        
        copyButtons.forEach(button => {
            button.addEventListener('click', () => {
                const textToCopy = button.getAttribute('data-copy');
                this.copyToClipboard(textToCopy, 'Kopyalandı!');
            });
        });
    }

    /**
     * Copy text to clipboard
     */
    async copyToClipboard(text, successMessage = 'Kopyalandı!') {
        try {
            if (navigator.clipboard && window.isSecureContext) {
                await navigator.clipboard.writeText(text);
            } else {
                // Fallback for older browsers
                const textArea = document.createElement('textarea');
                textArea.value = text;
                textArea.style.position = 'fixed';
                textArea.style.left = '-999999px';
                textArea.style.top = '-999999px';
                document.body.appendChild(textArea);
                textArea.focus();
                textArea.select();
                document.execCommand('copy');
                textArea.remove();
            }
            
            this.showAlert(successMessage, 'success');
        } catch (error) {
            console.error('Copy to clipboard failed:', error);
            this.showAlert('Kopyalama işlemi başarısız oldu.', 'error');
        }
    }

    /**
     * Show alert message
     */
    showAlert(message, type = 'success') {
        const alertId = type === 'success' ? 'successMessage' : 'errorMessage';
        const alert = document.getElementById(alertId);
        const textElement = alert?.querySelector('.alert-text');

        if (!alert || !textElement) return;

        textElement.textContent = message;
        alert.style.display = 'flex';
        setTimeout(() => alert.classList.add('show'), 10);

        // Auto hide after 5 seconds
        setTimeout(() => {
            this.closeAlert(alertId);
        }, 5000);
    }

    /**
     * Close alert
     */
    closeAlert(alertId) {
        const alert = document.getElementById(alertId);
        if (!alert) return;

        alert.classList.remove('show');
        setTimeout(() => {
            alert.style.display = 'none';
        }, 300);
    }

    /**
     * Handle window resize
     */
    handleResize() {
        // Close any open modals on mobile
        if (window.innerWidth < 768) {
            const orderModal = document.getElementById('orderChangeModal');
            if (orderModal?.classList.contains('show')) {
                this.closeOrderModal();
            }
        }
    }

    /**
     * Setup keyboard shortcuts
     */
    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            // Ctrl/Cmd + E for edit
            if ((e.ctrlKey || e.metaKey) && e.key === 'e') {
                e.preventDefault();
                const editBtn = document.querySelector('a[href*="/duzenle"]');
                if (editBtn) {
                    editBtn.click();
                }
            }
            
            // Ctrl/Cmd + O for order change
            if ((e.ctrlKey || e.metaKey) && e.key === 'o') {
                e.preventDefault();
                this.showOrderChangeModal();
            }
            
            // Ctrl/Cmd + D for delete
            if ((e.ctrlKey || e.metaKey) && e.key === 'd') {
                e.preventDefault();
                const deleteBtn = document.getElementById('deleteHeaderButtonBtn') || document.getElementById('deleteIndividualHeaderButtonBtn');
                if (deleteBtn) {
                    deleteBtn.click();
                }
            }
            
            // Ctrl/Cmd + B for back to list
            if ((e.ctrlKey || e.metaKey) && e.key === 'b') {
                e.preventDefault();
                window.location.href = '/panel/header-tuslari';
            }
        });
    }
}

/**
 * Global functions for template usage
 */
window.closeDeleteModal = function() {
    const modal = document.getElementById('deleteModal');
    modal.classList.remove('show');
    setTimeout(() => {
        modal.style.display = 'none';
    }, 300);
    document.body.style.overflow = '';
};

window.closeOrderModal = function() {
    const modal = document.getElementById('orderChangeModal');
    modal.classList.remove('show');
    setTimeout(() => {
        modal.style.display = 'none';
    }, 300);
    document.body.style.overflow = '';
};

window.closeAlert = function(alertId) {
    const alert = document.getElementById(alertId);
    alert.classList.remove('show');
    setTimeout(() => {
        alert.style.display = 'none';
    }, 300);
};

/**
 * Utility functions
 */
function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('tr-TR', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.headerButtonViewManager = new HeaderButtonViewManager();
    
    // Add CSS for animations
    const style = document.createElement('style');
    style.textContent = `
        .content-card {
            opacity: 1;
            transform: translateY(0);
            transition: all 0.6s ease;
        }
        
        .content-card.animate-in {
            opacity: 1;
            transform: translateY(0);
        }
        
        .loading-overlay {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(255, 255, 255, 0.8);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 9999;
        }
        
        .loading-spinner {
            width: 40px;
            height: 40px;
            border: 4px solid var(--border-color);
            border-top: 4px solid var(--primary-color);
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
    `;
    document.head.appendChild(style);
});

// Export for potential external use
window.HeaderButtonViewManager = HeaderButtonViewManager;
