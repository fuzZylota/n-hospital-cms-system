/**
 * Anlasmali Kurumlar (Contract Partners) Page JavaScript
 * Handles table interactions, filtering, pagination, and CRUD operations
 */

class AnlasmaliKurumlarListManager {
    constructor() {
        this.data = window.pageData || {};
        this.currentView = 'table';
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupDropdowns();
        this.setupSearch();
        this.setupFilters();
        this.setupPagination();
        this.setupViewToggle();
        this.setupTableSorting();
        this.updateStatistics();
        this.bindRefreshButton();
    }

    /**
     * Setup all event listeners
     */
    setupEventListeners() {
        // Close dropdowns when clicking outside
        document.addEventListener('click', (e) => {
            if (!e.target.closest('.actions-wrapper')) {
                this.closeAllDropdowns();
            }
            
            // Close modal when clicking outside
            if (e.target.classList.contains('modal')) {
                this.closeDeleteModal();
            }
        });

        // Handle escape key
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.closeAllDropdowns();
                const modal = document.querySelector('.modal.show');
                if (modal) {
                    this.closeDeleteModal();
                }
            }
        });

        // Handle window resize for responsive behavior
        window.addEventListener('resize', () => {
            this.handleResize();
        });
    }

    /**
     * Setup action dropdowns
     */
    setupDropdowns() {
        const actionTriggers = document.querySelectorAll('.action-trigger');
        
        actionTriggers.forEach(trigger => {
            trigger.addEventListener('click', (e) => {
                e.stopPropagation();
                const dropdown = trigger.parentNode.querySelector('.action-dropdown');
                const isOpen = dropdown.classList.contains('show');
                
                // Close all dropdowns first
                this.closeAllDropdowns();
                
                // Toggle current dropdown
                if (!isOpen) {
                    dropdown.classList.add('show');
                }
            });
        });

        // Setup delete buttons
        const deleteButtons = document.querySelectorAll('.action-item.delete');
        deleteButtons.forEach(button => {
            button.addEventListener('click', (e) => {
                e.preventDefault();
                const id = button.getAttribute('data-id');
                const name = button.getAttribute('data-name');
                this.showDeleteConfirmation(id, name);
            });
        });
    }

    /**
     * Close all open dropdowns
     */
    closeAllDropdowns() {
        const openDropdowns = document.querySelectorAll('.action-dropdown.show');
        openDropdowns.forEach(dropdown => {
            dropdown.classList.remove('show');
        });
    }

    /**
     * Setup search functionality
     */
    setupSearch() {
        const searchInput = document.getElementById('searchInput');
        const searchClear = document.getElementById('searchClear');

        if (!searchInput) return;

        // Only show/hide clear button, no automatic search
        searchInput.addEventListener('input', (e) => {
            const value = e.target.value.trim();
            
            // Show/hide clear button
            if (value) {
                searchClear.style.display = 'block';
            } else {
                searchClear.style.display = 'none';
            }
        });

        // Clear search
        searchClear.addEventListener('click', () => {
            searchInput.value = '';
            searchClear.style.display = 'none';
        });
    }

    /**
     * Setup filter dropdowns
     */
    setupFilters() {
        // Disabled automatic filtering - now handled by manual filter button
        // Filters are only applied when the filter button is clicked
    }

    /**
     * Setup table column sorting
     */
    setupTableSorting() {
        // Sorting is handled by the template filter button
        // No complex client-side sorting needed
    }

    /**
     * Setup view toggle (table/grid)
     */
    setupViewToggle() {
        const viewButtons = document.querySelectorAll('.view-btn');
        const tableView = document.getElementById('tableView');
        const gridView = document.getElementById('gridView');

        viewButtons.forEach(button => {
            button.addEventListener('click', () => {
                const view = button.getAttribute('data-view');
                
                // Update active button
                viewButtons.forEach(btn => btn.classList.remove('active'));
                button.classList.add('active');
                
                // Toggle views
                if (view === 'grid') {
                    tableView.style.display = 'none';
                    gridView.style.display = 'block';
                    this.currentView = 'grid';
                } else {
                    tableView.style.display = 'block';
                    gridView.style.display = 'none';
                    this.currentView = 'table';
                }
                
                // Re-setup dropdowns for the active view
                setTimeout(() => {
                    this.setupDropdowns();
                }, 100);
            });
        });
    }

    /**
     * Setup pagination
     */
    setupPagination() {
        // Pagination is handled by the template
        // No complex client-side pagination needed
    }

    /**
     * Filter and reload data
     */
    async filterAndReload() {
        // Simple reload - filtering is handled by the template
        window.location.reload();
    }

    /**
     * Update statistics cards
     */
    updateStatistics() {
        // Statistics are handled by the template
        // No complex client-side statistics needed
    }

    /**
     * Show delete confirmation modal
     */
    showDeleteConfirmation(id, name) {
        const modal = document.getElementById('deleteModal');
        const nameSpan = document.getElementById('deleteItemName');
        const confirmBtn = document.getElementById('confirmDeleteBtn');

        if (nameSpan) nameSpan.textContent = name;
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);

        // Remove existing listeners
        const newConfirmBtn = confirmBtn.cloneNode(true);
        confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);

        // Add new listener
        newConfirmBtn.addEventListener('click', () => {
            this.deleteAnlasmaliKurum(id);
        });

        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Delete contract partner
     */
    async deleteAnlasmaliKurum(id) {
        const modal = document.getElementById('deleteModal');
        const confirmBtn = modal.querySelector('#confirmDeleteBtn');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');

        if (!confirmBtn || !btnText || !btnLoader) return;

        try {
            // Show loading state
            confirmBtn.disabled = true;
            btnText.style.opacity = '0';
            btnLoader.style.display = 'block';

            const response = await fetch(`/backend/anlasmali-kurum/${id}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const responseData = await response.json().catch(() => ({}));

            if (responseData.status === 201) {
                this.closeDeleteModal();
                this.showAlert('Anlaşmalı kurum başarıyla silindi.', 'success');
                
                // Remove row from table/card from grid
                const tableRow = document.querySelector(`tr[data-id="${id}"]`);
                const gridCard = document.querySelector(`.anlasmali-kurum-card[data-id="${id}"]`);
                
                if (tableRow) {
                    tableRow.style.opacity = '0';
                    setTimeout(() => tableRow.remove(), 300);
                }
                if (gridCard) {
                    gridCard.style.opacity = '0';
                    setTimeout(() => gridCard.remove(), 300);
                }

                // Update count and statistics
                this.updateStatistics();

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
     * Show alert message
     */
    showAlert(message, type = 'success') {
        // Create alert if it doesn't exist
        let alertId = type === 'success' ? 'successMessage' : 'errorMessage';
        let alert = document.getElementById(alertId);
        
        if (!alert) {
            alert = this.createAlert(alertId, type);
        }

        const textElement = alert.querySelector('.alert-text');
        if (textElement) {
            textElement.textContent = message;
        }

        alert.style.display = 'flex';
        setTimeout(() => alert.classList.add('show'), 10);

        // Auto hide after 5 seconds
        setTimeout(() => {
            this.closeAlert(alertId);
        }, 5000);
    }

    /**
     * Create alert element
     */
    createAlert(id, type) {
        const alert = document.createElement('div');
        alert.id = id;
        alert.className = `alert alert-${type === 'success' ? 'success' : 'error'}`;
        alert.style.display = 'none';
        
        const icon = type === 'success' ? 'fas fa-check-circle' : 'fas fa-exclamation-circle';
        alert.innerHTML = `
            <i class="${icon}"></i>
            <span class="alert-text"></span>
            <button class="alert-close" onclick="closeAlert('${id}')">
                <i class="fas fa-times"></i>
            </button>
        `;
        
        document.body.appendChild(alert);
        return alert;
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
        // Close dropdowns on resize
        this.closeAllDropdowns();
    }

    /**
     * Bind refresh button
     */
    bindRefreshButton() {
        const refreshBtn = document.getElementById('refreshBtn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                window.location.reload();
            });
        }
    }

    /**
     * Export functionality
     */
    setupExport() {
        // Export is handled by the template
        // No complex client-side export needed
    }
}

/**
 * Global functions for template usage
 */
window.closeDeleteModal = function() {
    const modal = document.getElementById('deleteModal');
    if (modal) {
        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
        document.body.style.overflow = '';
    }
};

window.closeAlert = function(alertId) {
    const alert = document.getElementById(alertId);
    if (alert) {
        alert.classList.remove('show');
        setTimeout(() => {
            alert.style.display = 'none';
        }, 300);
    }
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

function truncateText(text, maxLength = 50) {
    if (text.length <= maxLength) return text;
    return text.substring(0, maxLength) + '...';
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
    new AnlasmaliKurumlarListManager();
});

// Export for potential external use
window.AnlasmaliKurumlarListManager = AnlasmaliKurumlarListManager;
