/**
 * Haberler (News List) Page JavaScript
 * Handles table interactions, filtering, pagination, and CRUD operations
 */

class HaberlerListManager {
    constructor() {
        this.data = window.pageData || {};
        this.currentPage = this.data.currentPage || 1;
        this.perPage = this.data.perPage || 10;
        this.totalCount = this.data.count || 0;
        this.currentSort = { column: 'updated_at', order: 'desc' };
        this.currentFilters = { search: '', status: '', category: '', featured: '' };
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
        this.populateCategoryFilter();
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

        // Only show/hide clear button, no automatic filtering
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
     * Populate category filter with unique categories from data
     */
    populateCategoryFilter() {
        const categoryFilter = document.getElementById('categoryFilter');
        if (!categoryFilter) return;

        const haberler = this.data.haberler || [];
        const categories = [...new Set(haberler.map(haber => haber.category).filter(category => category))];
        categories.sort();

        categories.forEach(category => {
            const option = document.createElement('option');
            option.value = category;
            option.textContent = category;
            categoryFilter.appendChild(option);
        });
    }

    /**
     * Setup table column sorting
     */
    setupTableSorting() {
        // Disabled automatic table sorting - now handled by manual filter button
        // Sorting is only applied when the filter button is clicked
    }

    /**
     * Update sort header indicators
     */
    updateSortHeaders() {
        const headers = document.querySelectorAll('.data-table th.sortable');
        headers.forEach(header => {
            header.classList.remove('asc', 'desc');
            if (header.getAttribute('data-column') === this.currentSort.column) {
                header.classList.add(this.currentSort.order);
            }
        });
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
        this.renderPagination();
        
        // Per page selector
        const perPageSelect = document.getElementById('perPage');
        if (perPageSelect) {
            perPageSelect.value = this.perPage;
            perPageSelect.addEventListener('change', (e) => {
                this.perPage = parseInt(e.target.value);
                this.currentPage = 1;
                this.filterAndReload();
            });
        }
    }

    /**
     * Render pagination controls
     */
    renderPagination() {
        const paginationContainer = document.getElementById('pagination');
        if (!paginationContainer || this.totalCount === 0) return;

        const totalPages = Math.ceil(this.totalCount / this.perPage);
        const currentPage = this.currentPage;
        
        let paginationHTML = '';

        // Previous button
        paginationHTML += `
            <button class="pagination-btn ${currentPage === 1 ? 'disabled' : ''}" 
                    data-page="${currentPage - 1}" ${currentPage === 1 ? 'disabled' : ''}>
                <i class="fas fa-chevron-left"></i>
            </button>
        `;

        // Page numbers
        const startPage = Math.max(1, currentPage - 2);
        const endPage = Math.min(totalPages, currentPage + 2);

        if (startPage > 1) {
            paginationHTML += `<button class="pagination-btn" data-page="1">1</button>`;
            if (startPage > 2) {
                paginationHTML += `<span class="pagination-ellipsis">...</span>`;
            }
        }

        for (let i = startPage; i <= endPage; i++) {
            paginationHTML += `
                <button class="pagination-btn ${i === currentPage ? 'active' : ''}" data-page="${i}">
                    ${i}
                </button>
            `;
        }

        if (endPage < totalPages) {
            if (endPage < totalPages - 1) {
                paginationHTML += `<span class="pagination-ellipsis">...</span>`;
            }
            paginationHTML += `<button class="pagination-btn" data-page="${totalPages}">${totalPages}</button>`;
        }

        // Next button
        paginationHTML += `
            <button class="pagination-btn ${currentPage === totalPages ? 'disabled' : ''}" 
                    data-page="${currentPage + 1}" ${currentPage === totalPages ? 'disabled' : ''}>
                <i class="fas fa-chevron-right"></i>
            </button>
        `;

        paginationContainer.innerHTML = paginationHTML;

        // Add click handlers
        const paginationButtons = paginationContainer.querySelectorAll('.pagination-btn:not(.disabled)');
        paginationButtons.forEach(button => {
            button.addEventListener('click', () => {
                const page = parseInt(button.getAttribute('data-page'));
                if (page && page !== this.currentPage) {
                    this.currentPage = page;
                    this.filterAndReload();
                }
            });
        });
    }

    /**
     * Filter and reload data
     */
    async filterAndReload() {
        try {
            // Build query parameters
            const params = new URLSearchParams();
            params.set('page', this.currentPage);
            params.set('per_page', this.perPage);
            params.set('sort_by', this.currentSort.column);
            params.set('sort_order', this.currentSort.order);
            
            if (this.currentFilters.search) {
                params.set('search', this.currentFilters.search);
            }
            if (this.currentFilters.status) {
                params.set('status', this.currentFilters.status);
            }
            if (this.currentFilters.category) {
                params.set('category', this.currentFilters.category);
            }
            if (this.currentFilters.featured) {
                params.set('featured', this.currentFilters.featured);
            }

            // Reload page with new parameters
            window.location.href = `/panel/haberler?${params.toString()}`;
        } catch (error) {
            console.error('Filter and reload error:', error);
            this.showAlert('Filtreler uygulanırken bir hata oluştu.', 'error');
        }
    }

    /**
     * Update statistics cards
     */
    updateStatistics() {
        const haberler = this.data.haberler || [];
        
        const publishedCount = haberler.filter(haber => haber.is_published).length;
        const draftCount = haberler.length - publishedCount;
        const featuredCount = haberler.filter(haber => haber.is_featured).length;

        // Update DOM
        const publishedCountEl = document.getElementById('publishedCount');
        const draftCountEl = document.getElementById('draftCount');
        const featuredCountEl = document.getElementById('featuredCount');

        if (publishedCountEl) publishedCountEl.textContent = publishedCount;
        if (draftCountEl) draftCountEl.textContent = draftCount;
        if (featuredCountEl) featuredCountEl.textContent = featuredCount;
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
            this.deleteHaber(id);
        });

        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Delete news
     */
    async deleteHaber(id) {
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

            const response = await fetch(`/backend/news/${id}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const responseData = await response.json().catch(() => ({}));

            if (responseData.status === 201) {
                this.closeDeleteModal();
                this.showAlert('Haber başarıyla silindi.', 'success');
                
                // Remove row from table/card from grid
                const tableRow = document.querySelector(`tr[data-id="${id}"]`);
                const gridCard = document.querySelector(`.haber-card[data-id="${id}"]`);
                
                if (tableRow) {
                    tableRow.style.opacity = '0';
                    setTimeout(() => tableRow.remove(), 300);
                }
                if (gridCard) {
                    gridCard.style.opacity = '0';
                    setTimeout(() => gridCard.remove(), 300);
                }

                // Update count and statistics
                this.totalCount--;
                this.updateStatistics();
                
                // Reload if current page becomes empty
                setTimeout(() => {
                    const remainingRows = document.querySelectorAll('tbody tr[data-id]').length;
                    if (remainingRows === 0 && this.currentPage > 1) {
                        this.currentPage--;
                        this.filterAndReload();
                    }
                }, 500);

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
        
        // Update pagination if needed
        if (window.innerWidth < 768) {
            // Mobile optimizations
            this.setupMobileOptimizations();
        }
    }

    /**
     * Setup mobile optimizations
     */
    setupMobileOptimizations() {
        // Force grid view on very small screens
        if (window.innerWidth < 480 && this.currentView === 'table') {
            const gridViewBtn = document.querySelector('.view-btn[data-view="grid"]');
            if (gridViewBtn) {
                gridViewBtn.click();
            }
        }
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
        const exportBtn = document.getElementById('exportBtn');
        if (exportBtn) {
            exportBtn.addEventListener('click', () => {
                this.exportData();
            });
        }
    }

    /**
     * Export data to CSV
     */
    async exportData() {
        try {
            const response = await fetch('/panel/haberler/export', {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            if (response.ok) {
                const blob = await response.blob();
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = `haberler-${new Date().toISOString().split('T')[0]}.csv`;
                document.body.appendChild(a);
                a.click();
                window.URL.revokeObjectURL(url);
                document.body.removeChild(a);
                
                this.showAlert('Veriler başarıyla dışa aktarıldı.', 'success');
            } else {
                this.showAlert('Dışa aktarma sırasında bir hata oluştu.', 'error');
            }
        } catch (error) {
            console.error('Export error:', error);
            this.showAlert('Dışa aktarma sırasında bir hata oluştu.', 'error');
        }
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
    new HaberlerListManager();
    
    // Setup export functionality
    const exportBtn = document.getElementById('exportBtn');
    if (exportBtn) {
        exportBtn.addEventListener('click', () => {
            // Simple CSV export functionality
            const table = document.querySelector('.data-table');
            if (table) {
                exportTableToCSV(table, 'haberler.csv');
            }
        });
    }
});

/**
 * Simple CSV export function
 */
function exportTableToCSV(table, filename) {
    const csv = [];
    const rows = table.querySelectorAll('tr');
    
    for (let i = 0; i < rows.length; i++) {
        const row = [];
        const cols = rows[i].querySelectorAll('td, th');
        
        for (let j = 0; j < cols.length - 1; j++) { // Exclude actions column
            let cellText = cols[j].innerText.replace(/\s+/g, ' ').trim();
            cellText = cellText.replace(/"/g, '""'); // Escape quotes
            row.push(`"${cellText}"`);
        }
        
        csv.push(row.join(','));
    }
    
    const csvContent = csv.join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    
    if (link.download !== undefined) {
        const url = URL.createObjectURL(blob);
        link.setAttribute('href', url);
        link.setAttribute('download', filename);
        link.style.visibility = 'hidden';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    }
}

// Export for potential external use
window.HaberlerListManager = HaberlerListManager;
