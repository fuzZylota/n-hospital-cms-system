/**
 * Randevu Talepleri (Appointment Requests) Page JavaScript
 * Handles table interactions, filtering, pagination, and CRUD operations
 */

class RandevuTalepleriListManager {
    constructor() {
        this.data = window.pageData || {};
        this.currentPage = this.data.currentPage || 1;
        this.perPage = this.data.perPage || 10;
        this.totalCount = this.data.count || 0;
        this.currentSort = { column: 'created_at', order: 'desc' };
        this.currentFilters = { search: '', date: '' };
        this.currentView = 'table';
        this.dropdownHandlers = new Map(); // Store handlers for cleanup
        
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
        //this.updateStatistics();
        this.bindRefreshButton();
    }

    /**
     * Setup all event listeners
     */
    setupEventListeners() {
        // Close dropdowns when clicking outside
        document.addEventListener('click', (e) => {
            // Don't close if clicking on toggle-status button or its parent dropdown
            if (!e.target.closest('.actions-wrapper') && !e.target.closest('.action-item.toggle-status')) {
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
        // Remove old event listeners first
        this.removeDropdownHandlers();
        
        // Only setup dropdowns for the currently visible view
        const tableView = document.getElementById('tableView');
        const gridView = document.getElementById('gridView');
        const isTableViewVisible = tableView && tableView.style.display !== 'none';
        const isGridViewVisible = gridView && gridView.style.display !== 'none';
        
        // Get action triggers only from visible view
        let actionTriggers = [];
        if (isTableViewVisible) {
            actionTriggers = Array.from(tableView.querySelectorAll('.action-trigger'));
        } else if (isGridViewVisible) {
            actionTriggers = Array.from(gridView.querySelectorAll('.action-trigger'));
        }
        
        actionTriggers.forEach(trigger => {
            const handler = (e) => {
                e.stopPropagation();
                const dropdown = trigger.parentNode.querySelector('.action-dropdown');
                if (!dropdown) return;
                
                const isOpen = dropdown.classList.contains('show');
                
                // Close all dropdowns first
                this.closeAllDropdowns();
                
                // Toggle current dropdown
                if (!isOpen) {
                    dropdown.classList.add('show');
                }
            };
            
            trigger.addEventListener('click', handler);
            this.dropdownHandlers.set(trigger, handler);
        });

        // Setup delete buttons (only from visible view)
        let deleteButtons = [];
        if (isTableViewVisible) {
            deleteButtons = Array.from(tableView.querySelectorAll('.action-item.delete'));
        } else if (isGridViewVisible) {
            deleteButtons = Array.from(gridView.querySelectorAll('.action-item.delete'));
        }
        
        deleteButtons.forEach(button => {
            const handler = (e) => {
                e.preventDefault();
                e.stopPropagation();
                const id = button.getAttribute('data-id');
                const name = button.getAttribute('data-name');
                this.showDeleteConfirmation(id, name);
            };
            
            button.addEventListener('click', handler);
            this.dropdownHandlers.set(button, handler);
        });

        // Setup toggle status buttons (only from visible view)
        let toggleStatusButtons = [];
        if (isTableViewVisible) {
            toggleStatusButtons = Array.from(tableView.querySelectorAll('.action-item.toggle-status'));
        } else if (isGridViewVisible) {
            toggleStatusButtons = Array.from(gridView.querySelectorAll('.action-item.toggle-status'));
        }
        
        toggleStatusButtons.forEach(button => {
            const handler = (e) => {
                e.preventDefault();
                e.stopPropagation();
                const id = button.getAttribute('data-id');
                const status = button.getAttribute('data-status');
                this.toggleRandevuTalepStatus(id, status, button);
            };
            
            button.addEventListener('click', handler);
            this.dropdownHandlers.set(button, handler);
        });
    }

    /**
     * Remove all dropdown event handlers
     */
    removeDropdownHandlers() {
        this.dropdownHandlers.forEach((handler, element) => {
            element.removeEventListener('click', handler);
        });
        this.dropdownHandlers.clear();
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
     * Setup table column sorting
     */
    setupTableSorting() {
        const urlParams = new URLSearchParams(window.location.search);
        const currentSortBy = urlParams.get('sort_by') || 'created_at';
        const currentSortOrder = (urlParams.get('sort_order') || 'DESC').toUpperCase();

        const headers = document.querySelectorAll('.data-table th.sortable');
        headers.forEach(header => {
            const column = header.getAttribute('data-column');
            if (column === currentSortBy) {
                header.classList.add(currentSortOrder === 'ASC' ? 'asc' : 'desc');
            }
            header.addEventListener('click', () => {
                let newOrder;
                if (column === currentSortBy) {
                    newOrder = currentSortOrder === 'ASC' ? 'DESC' : 'ASC';
                } else {
                    newOrder = 'DESC';
                }
                const params = new URLSearchParams(window.location.search);
                params.set('sort_by', column);
                params.set('sort_order', newOrder);
                params.set('page', '1');
                window.location.href = window.location.pathname + '?' + params.toString();
            });
        });
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
                // Use requestAnimationFrame to ensure DOM is updated
                requestAnimationFrame(() => {
                    this.setupDropdowns();
                });
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
            if (this.currentFilters.date) {
                params.set('date', this.currentFilters.date);
            }

            // Reload page with new parameters
            window.location.href = `/panel/randevu-talepleri?${params.toString()}`;
        } catch (error) {
            console.error('Filter and reload error:', error);
            this.showAlert('Filtreler uygulanırken bir hata oluştu.', 'error');
        }
    }

    /**
     * Update statistics cards
     */
    updateStatistics() {
        const randevuTalepleri = this.data.randevuTalepleri || [];
        const now = new Date();
        const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
        const weekStart = new Date(today.getTime() - (today.getDay() * 24 * 60 * 60 * 1000));
        const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);
        
        const todayCount = randevuTalepleri.filter(talep => {
            const talepDate = new Date(talep.created_at);
            return talepDate >= today;
        }).length;
        
        const weekCount = randevuTalepleri.filter(talep => {
            const talepDate = new Date(talep.created_at);
            return talepDate >= weekStart;
        }).length;
        
        const monthCount = randevuTalepleri.filter(talep => {
            const talepDate = new Date(talep.created_at);
            return talepDate >= monthStart;
        }).length;

        // Update DOM
        const todayCountEl = document.getElementById('todayCount');
        const weekCountEl = document.getElementById('weekCount');
        const monthCountEl = document.getElementById('monthCount');

        if (todayCountEl) todayCountEl.textContent = todayCount;
        if (weekCountEl) weekCountEl.textContent = weekCount;
        if (monthCountEl) monthCountEl.textContent = monthCount;
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
            this.deleteRandevuTalep(id);
        });

        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Delete appointment request
     */
    async deleteRandevuTalep(id) {
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

            const response = await fetch(`/backend/randevu-request/${id}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const responseData = await response.json().catch(() => ({}));

            if (responseData.status === 201) {
                this.closeDeleteModal();
                this.showAlert('Randevu talebi başarıyla silindi.', 'success');
                
                // Remove row from table/card from grid
                const tableRow = document.querySelector(`tr[data-id="${id}"]`);
                const gridCard = document.querySelector(`.randevu-talebi-card[data-id="${id}"]`);
                
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
                
                // Reload if current page becomes empty
                setTimeout(() => {
                    const remainingRows = document.querySelectorAll('tbody tr[data-id]').length;
                    if (remainingRows === 0 && this.currentPage > 1) {
                        this.currentPage--;
                        this.filterAndReload();
                    }
                }, 500);

            } else {
                this.closeDeleteModal();
                this.showAlert(responseData.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            }
        } catch (error) {
            console.error('Delete error:', error);
            this.closeDeleteModal();
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
     * Get status text in Turkish
     */
    getStatusText(status) {
        const statusMap = {
            'yeni': 'Yeni',
            'randevu-verildi': 'Randevu Verildi',
            'randevu-verilemedi': 'Randevu Verilemedi',
            'ulasilamadi': 'Ulaşılamadı',
            'gelmedi': 'Gelmedi',
            'hasta-vazgecti': 'Hasta Vazgeçti'
        };
        return statusMap[status] || status;
    }

    /**
     * Toggle randevu talep status
     */
    async toggleRandevuTalepStatus(rrid, currentStatus, button) {
        if (!rrid || !currentStatus) return;

        const originalText = button.querySelector('span').textContent;
        const icon = button.querySelector('i');
        const originalIconClass = icon.className;

        try {
            // Show loading state
            button.disabled = true;
            icon.className = 'fas fa-spinner fa-spin';
            button.querySelector('span').textContent = 'Değiştiriliyor...';

            const response = await fetch(`/backend/randevu-request/${rrid}/toggle-status`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                },
                body: JSON.stringify({
                    status: currentStatus
                })
            });

            const responseData = await response.json().catch(() => ({}));

            if (responseData.status === 201 && responseData.new_status) {
                const newStatus = responseData.new_status;

                // Update status badge in table view
                const tableRow = document.querySelector(`tr[data-id="${rrid}"]`);
                if (tableRow) {
                    const statusCell = tableRow.querySelector('.status-badge');
                    if (statusCell) {
                        statusCell.className = `status-badge status-badge--${newStatus}`;
                        statusCell.textContent = this.getStatusText(newStatus);
                    }
                }

                // Update status badge in grid view
                const gridCard = document.querySelector(`.randevu-talebi-card[data-id="${rrid}"]`);
                if (gridCard) {
                    const statusBadge = gridCard.querySelector('.status-badge');
                    if (statusBadge) {
                        statusBadge.className = `status-badge status-badge--${newStatus}`;
                        statusBadge.textContent = this.getStatusText(newStatus);
                    }
                }

                // Update button data-status attribute
                button.setAttribute('data-status', newStatus);

                // Update data in pageData if exists
                if (this.data && this.data.randevuTalepleri) {
                    const talep = this.data.randevuTalepleri.find(t => t.rrid === rrid);
                    if (talep) {
                        talep.status = newStatus;
                    }
                }

            } else {
                this.showAlert(responseData.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            }
        } catch (error) {
            console.error('Toggle status error:', error);
            this.showAlert('Bağlantı hatası oluştu.', 'error');
        } finally {
            // Reset button state
            button.disabled = false;
            icon.className = originalIconClass;
            button.querySelector('span').textContent = originalText;
        }
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
            const response = await fetch('/panel/randevu-talepleri/export', {
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
                a.download = `randevu-talepleri-${new Date().toISOString().split('T')[0]}.csv`;
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
// Hasta tarafindan girilen (isim, telefon, mesaj vb.) her deger HTML'e
// basilmadan once bu fonksiyondan gecirilmeli - anonim herkes randevu
// formunu doldurabildigi icin, escape edilmezse admin panelinde
// calisan bir stored XSS'e donusuyordu (kullanici raporuyla bulundu).
function escapeHtml(value) {
    if (value === null || value === undefined) return '';
    return String(value)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

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
    window._rtManager = new RandevuTalepleriListManager();
    
    // Export modal ile handle ediliyor
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
window.RandevuTalepleriListManager = RandevuTalepleriListManager;

// Yeni talep polling
(function() {
    const POLL_INTERVAL = 30000;
    let knownRrids = new Set();
    window._knownRrids = knownRrids;

    window._playBellSound = playBellSound;
    function playBellSound() {
        try {
            const ctx = new (window.AudioContext || window.webkitAudioContext)();
            function schoolBell(startTime) {
                const frequencies = [1047, 1319, 1047, 1319];
                frequencies.forEach((freq, i) => {
                    const osc = ctx.createOscillator();
                    const gain = ctx.createGain();
                    const t = startTime + i * 0.12;
                    osc.connect(gain);
                    gain.connect(ctx.destination);
                    osc.type = 'square';
                    osc.frequency.setValueAtTime(freq, t);
                    gain.gain.setValueAtTime(0.15, t);
                    gain.gain.exponentialRampToValueAtTime(0.001, t + 0.1);
                    osc.start(t);
                    osc.stop(t + 0.1);
                });
            }
            schoolBell(ctx.currentTime);
            schoolBell(ctx.currentTime + 0.6);
        } catch(e) {}
    }

    function statusLabel(status) {
        const map = {
            'yeni': 'Yeni',
            'randevu-verildi': 'Randevu Verildi',
            'randevu-verilemedi': 'Randevu Verilemedi',
            'ulasilamadi': 'Ulaşılamadı',
            'gelmedi': 'Gelmedi',
            'hasta-vazgecti': 'Hasta Vazgeçti',
            'hasta-arandi': 'Hasta Arandı'
        };
        return map[status] || status;
    }

    function formatDate(dateStr) {
        if (!dateStr || dateStr === '0001-01-01T00:00:00Z' || dateStr === '0001-01-01 00:00:00+00:00') return '-';
        // PostgreSQL formatlarını normalize et
        // DB +03:00 olarak kaydediyor, direkt parse et
        const normalized = dateStr.replace(' ', 'T');
        const d = new Date(normalized);
        if (isNaN(d.getTime())) return dateStr;
        const pad = n => String(n).padStart(2, '0');
        // getUTC + 3 saat = Istanbul saati (DB zaten +03 ile geliyor)
        const utcMs = d.getTime();
        const istMs = utcMs + (3 * 60 * 60 * 1000);
        const ist = new Date(istMs);
        return pad(ist.getUTCDate()) + '.' + pad(ist.getUTCMonth()+1) + '.' + ist.getUTCFullYear() + ' ' + pad(ist.getUTCHours()) + ':' + pad(ist.getUTCMinutes());
    }

    function clearNewHighlights() {
        document.querySelectorAll('tr.new-request-row').forEach(row => {
            row.classList.remove('new-request-row');
        });
    }

    function initKnownRrids() {
        document.querySelectorAll('tbody tr[data-id]').forEach(row => {
            knownRrids.add(row.getAttribute('data-id'));
        });
    }

    window._addNewRowToTable = addNewRowToTable;
    function addNewRowToTable(talep) {
        const tbody = document.querySelector('#randevuTalepleriTable tbody');
        if (!tbody) return;

        const safeFirstName = escapeHtml(talep.patient_first_name);
        const safeLastName = escapeHtml(talep.patient_last_name);
        const safePhone = escapeHtml(talep.patient_phone);
        const safeSubeName = escapeHtml(talep.sube_name);

        const tr = document.createElement('tr');
        tr.setAttribute('data-id', talep.rrid);
        tr.classList.add('new-request-row');
        tr.innerHTML = `
            <td>
                <div class="cell-content">
                    <div class="primary-text">${safeFirstName} ${safeLastName}</div>
                </div>
            </td>
            <td>
                <div class="cell-content">
                    ${talep.patient_phone
                        ? `<a href="tel:${safePhone}" class="phone-link">${safePhone}</a>`
                        : '<span class="text-muted">-</span>'}
                </div>
            </td>
            <td>
                <div class="cell-content">
                    <div class="primary-text">${formatDate(talep.created_at)}</div>
                </div>
            </td>
            <td>
                <div class="cell-content">
                    ${talep.sube_name
                        ? `<div class="primary-text">${safeSubeName}</div>`
                        : '<span class="text-muted">Belirtilmemiş</span>'}
                </div>
            </td>
            <td>
                <div class="cell-content">
                    ${talep.message
                        ? `<div class="primary-text truncate" title="${escapeHtml(talep.message)}">${escapeHtml(talep.message)}</div>`
                        : '<span class="text-muted">-</span>'}
                </div>
            </td>
            <td>
                <div class="cell-content">
                    <div class="status-inline">
                        <span class="status-badge status-badge--${talep.status}" data-status="${talep.status}">
                            ${statusLabel(talep.status)}
                        </span>
                        <span class="status-datetime">${formatDate(talep.created_at)}</span>
                    </div>
                </div>
            </td>
            <td>
                <div class="actions-wrapper">
                    <button class="action-trigger" data-id="${talep.rrid}">
                        <i class="fas fa-ellipsis-v"></i>
                    </button>
                    <div class="action-dropdown">
                        <a href="/panel/randevu-talepleri/${talep.rrid}" class="action-item view">
                            <i class="fas fa-eye"></i>
                            <span>Görüntüle</span>
                        </a>
                        <div class="action-divider"></div>
                        <button class="action-item toggle-status" data-id="${talep.rrid}" data-status="${talep.status}">
                            <i class="fas fa-sync-alt"></i>
                            <span>Durumu Değiştir</span>
                        </button>
                        <div class="action-divider"></div>
                        <button class="action-item delete" data-id="${talep.rrid}" data-name="${safeFirstName} ${safeLastName}">
                            <i class="fas fa-trash"></i>
                            <span>Sil</span>
                        </button>
                    </div>
                </div>
            </td>
        `;
        tbody.insertBefore(tr, tbody.firstChild);

        // Dropdown event listener ekle
        const trigger = tr.querySelector('.action-trigger');
        const dropdown = tr.querySelector('.action-dropdown');
        if (trigger && dropdown) {
            trigger.addEventListener('click', (e) => {
                e.stopPropagation();
                document.querySelectorAll('.action-dropdown.show').forEach(d => d.classList.remove('show'));
                dropdown.classList.add('show');
            });
        }
        const deleteBtn = tr.querySelector('.action-item.delete');
        if (deleteBtn) {
            deleteBtn.addEventListener('click', (e) => {
                e.preventDefault();
                const id = deleteBtn.getAttribute('data-id');
                const name = deleteBtn.getAttribute('data-name');
                if (window._rtManager) window._rtManager.showDeleteConfirmation(id, name);
            });
        }

        const toggleBtn = tr.querySelector('.action-item.toggle-status');
        if (toggleBtn) {
            toggleBtn.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                const id = toggleBtn.getAttribute('data-id');
                const status = toggleBtn.getAttribute('data-status');
                if (window._rtManager) window._rtManager.toggleRandevuTalepStatus(id, status, toggleBtn);
            });
        }
    }

    // Sayfa yüklenme zamanını since olarak kullan
    const pageSince = new Date().toISOString();
    window._pollNewRequests = async function pollNewRequests() {
        try {
            const res = await fetch('/panel/api/randevu-talepleri/latest?since=' + encodeURIComponent(pageSince), { credentials: 'include' });
            if (!res.ok) return;
            const json = await res.json();
            if (json.status === 200 && json.data && json.data.length > 0) {
                let hasNew = false;
                json.data.forEach(talep => {
                    if (!knownRrids.has(talep.rrid)) {
                        knownRrids.add(talep.rrid);
                        addNewRowToTable(talep);
                        hasNew = true;
                    }
                });
                if (hasNew) playBellSound();
            }
        } catch(e) {}
    }

    document.addEventListener('click', clearNewHighlights);
    initKnownRrids();
    setTimeout(() => {
        setInterval(window._pollNewRequests, POLL_INTERVAL);
    }, POLL_INTERVAL);
})();

// WebSocket ile gelen yeni talebi direkt ekle
window._addNewRandevuRow = function(talep) {
    if (typeof window._knownRrids !== 'undefined' && window._knownRrids.has(talep.rrid)) return;
    if (typeof window._knownRrids !== 'undefined') window._knownRrids.add(talep.rrid);
    if (typeof window._addNewRowToTable === 'function') {
        window._addNewRowToTable(talep);
        if (typeof window._playBellSound === 'function') window._playBellSound();
    } else if (typeof window._pollNewRequests === 'function') {
        window._pollNewRequests();
    }
};

// WebSocket ile gelen yeni talebi direkt ekle


// Export modal
window.openExportModal = function() {
    const modal = document.getElementById("exportModal");
    if (!modal) return;
    modal.style.display = 'flex';
    modal.style.alignItems = 'center';
    modal.style.justifyContent = 'center';
    setTimeout(function() {
        modal.onclick = function(e) { if (e.target === modal) window.closeExportModal(); };
    }, 50);
};
window.closeExportModal = function() {
    const modal = document.getElementById("exportModal");
    if (modal) { modal.style.display = 'none'; modal.onclick = null; }
};








window.doExport = async function() {
    const dateStart = document.getElementById('exportDateStart').value;
    const dateEnd = document.getElementById('exportDateEnd').value;
    const checkboxes = document.querySelectorAll('#exportModal input[type=checkbox]:checked');
    const statuses = Array.from(checkboxes).map(c => c.value).join(',');
    if (!dateStart || !dateEnd) {
        if (typeof window._rtManager !== 'undefined') {
            window._rtManager.showAlert('Lütfen tarih aralığı giriniz.', 'error');
        } else { alert('Lütfen tarih aralığı giriniz.'); }
        return;
    }
    if (dateStart > dateEnd) {
        if (typeof window._rtManager !== 'undefined') {
            window._rtManager.showAlert('Başlangıç tarihi bitiş tarihinden büyük olamaz.', 'error');
        } else { alert('Başlangıç tarihi bitiş tarihinden büyük olamaz.'); }
        return;
    }
    if (!statuses) {
        if (typeof window._rtManager !== 'undefined') {
            window._rtManager.showAlert('En az bir durum seçiniz.', 'error');
        } else { alert('En az bir durum seçiniz.'); }
        return;
    }
    let url = '/panel/randevu-talepleri/export?';
    if (dateStart) url += 'date_start=' + dateStart + '&';
    if (dateEnd) url += 'date_end=' + dateEnd + '&';
    url += 'statuses=' + encodeURIComponent(statuses);
    try {
        const res = await fetch(url, { credentials: 'include' });
        if (!res.ok) { alert('İndirme hatası oluştu.'); return; }
        const blob = await res.blob();
        const a = document.createElement('a');
        a.href = URL.createObjectURL(blob);
        a.download = 'randevu-talepleri-' + new Date().toISOString().split('T')[0] + '.xlsx';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        closeExportModal();
    } catch(e) {
        alert('İndirme hatası oluştu.');
    }
};

// Export butonunu modal'a bağla
document.addEventListener('DOMContentLoaded', function() {
    const exportBtn = document.getElementById('exportBtn');
    if (exportBtn) {
        exportBtn.onclick = function(e) { if(e) e.preventDefault(); openExportModal(); };
    }
});
