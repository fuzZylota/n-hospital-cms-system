// Randevular Page JavaScript
class RandevularListManager {
    constructor() {
        this.currentPage = window.pageData?.currentPage || 1;
        this.perPage = window.pageData?.perPage || 10;
        this.totalCount = window.pageData?.count || 0;
        this.randevular = window.pageData?.randevular || [];
        this.currentView = 'table';
        this.sortBy = 'appointment_date';
        this.sortOrder = 'desc';
        this.searchTerm = '';
        this.statusFilter = '';
        this.paymentFilter = '';
        this.deleteItemId = null;
        this.deleteItemName = '';
        
        this.init();
    }

    init() {
        this.bindEvents();
        this.updateStatistics();
        this.setupMobileOptimizations();
        this.bindRefreshButton();
        this.setupExport();
    }

    bindEvents() {
        // Search functionality
        const searchInput = document.getElementById('searchInput');
        const searchClear = document.getElementById('searchClear');
        
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                this.searchTerm = e.target.value;
                this.updateSearchClear();
            });
        }

        if (searchClear) {
            searchClear.addEventListener('click', () => {
                searchInput.value = '';
                this.searchTerm = '';
                this.updateSearchClear();
                this.filterAndReload();
            });
        }

        // Filter functionality
        const statusFilter = document.getElementById('statusFilter');
        const paymentFilter = document.getElementById('paymentFilter');
        const sortBy = document.getElementById('sortBy');
        const sortOrder = document.getElementById('sortOrder');

        // Disabled automatic filtering - now handled by manual filter button
        // Filters are only applied when the filter button is clicked

        // View toggle
        const viewButtons = document.querySelectorAll('.view-btn');
        viewButtons.forEach(btn => {
            btn.addEventListener('click', (e) => {
                const view = e.target.closest('.view-btn').dataset.view;
                this.toggleView(view);
            });
        });

        // Action dropdowns
        document.addEventListener('click', (e) => {
            if (e.target.closest('.action-trigger')) {
                e.preventDefault();
                e.stopPropagation();
                this.toggleActionDropdown(e.target.closest('.action-trigger'));
            } else {
                this.closeAllActionDropdowns();
            }
        });

        // Delete functionality
        document.addEventListener('click', (e) => {
            if (e.target.closest('.action-item.delete')) {
                e.preventDefault();
                const item = e.target.closest('.action-item.delete');
                const id = item.dataset.id;
                const name = item.dataset.name;
                this.showDeleteConfirmation(id, name);
            }
        });

        // Pagination
        document.addEventListener('click', (e) => {
            if (e.target.closest('.pagination a')) {
                e.preventDefault();
                const page = parseInt(e.target.closest('.pagination a').dataset.page);
                if (page && page !== this.currentPage) {
                    this.currentPage = page;
                    this.filterAndReload();
                }
            }
        });

        // Per page change
        const perPageSelect = document.getElementById('perPage');
        if (perPageSelect) {
            perPageSelect.addEventListener('change', (e) => {
                this.perPage = parseInt(e.target.value);
                this.currentPage = 1;
                this.filterAndReload();
            });
        }

        // Disabled automatic table sorting - now handled by manual filter button
        // Sorting is only applied when the filter button is clicked

        // Delete confirmation
        const confirmDeleteBtn = document.getElementById('confirmDeleteBtn');
        if (confirmDeleteBtn) {
            confirmDeleteBtn.addEventListener('click', () => {
                this.deleteRandevu();
            });
        }

        // Close modal on outside click
        const deleteModal = document.getElementById('deleteModal');
        if (deleteModal) {
            deleteModal.addEventListener('click', (e) => {
                if (e.target === deleteModal) {
                    this.closeDeleteModal();
                }
            });
        }
    }

    updateSearchClear() {
        const searchInput = document.getElementById('searchInput');
        const searchClear = document.getElementById('searchClear');
        
        if (searchInput && searchClear) {
            if (searchInput.value.trim()) {
                searchClear.style.display = 'block';
            } else {
                searchClear.style.display = 'none';
            }
        }
    }

    debounce(func, wait) {
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

    filterAndReload() {
        const params = new URLSearchParams();
        
        if (this.searchTerm) params.set('search', this.searchTerm);
        if (this.statusFilter) params.set('status', this.statusFilter);
        if (this.paymentFilter) params.set('payment_status', this.paymentFilter);
        if (this.sortBy) params.set('sort_by', this.sortBy);
        if (this.sortOrder) params.set('sort_order', this.sortOrder);
        if (this.currentPage > 1) params.set('page', this.currentPage);
        if (this.perPage !== 10) params.set('per_page', this.perPage);

        const queryString = params.toString();
        const url = queryString ? `/panel/randevular?${queryString}` : '/panel/randevular';
        
        window.location.href = url;
    }

    updateStatistics() {
        // Update pending count
        const pendingCount = this.randevular.filter(r => r.status === 'beklemede').length;
        const pendingElement = document.getElementById('pendingCount');
        if (pendingElement) {
            pendingElement.textContent = pendingCount;
        }

        // Update confirmed count
        const confirmedCount = this.randevular.filter(r => r.status === 'onaylandi').length;
        const confirmedElement = document.getElementById('confirmedCount');
        if (confirmedElement) {
            confirmedElement.textContent = confirmedCount;
        }

        // Update completed count
        const completedCount = this.randevular.filter(r => r.status === 'tamamlandi').length;
        const completedElement = document.getElementById('completedCount');
        if (completedElement) {
            completedElement.textContent = completedCount;
        }
    }

    toggleView(view) {
        this.currentView = view;
        
        // Update button states
        document.querySelectorAll('.view-btn').forEach(btn => {
            btn.classList.remove('active');
        });
        document.querySelector(`[data-view="${view}"]`).classList.add('active');
        
        // Show/hide views
        const tableView = document.getElementById('tableView');
        const gridView = document.getElementById('gridView');
        
        if (view === 'table') {
            tableView.style.display = 'block';
            gridView.style.display = 'none';
        } else {
            tableView.style.display = 'none';
            gridView.style.display = 'block';
        }
    }

    toggleActionDropdown(trigger) {
        this.closeAllActionDropdowns();
        const dropdown = trigger.nextElementSibling;
        if (dropdown) {
            dropdown.style.display = 'block';
        }
    }

    closeAllActionDropdowns() {
        document.querySelectorAll('.action-dropdown').forEach(dropdown => {
            dropdown.style.display = 'none';
        });
    }

    updateSortIcons() {
        document.querySelectorAll('.sort-icon').forEach(icon => {
            icon.style.opacity = '0.5';
        });
        
        const activeIcon = document.querySelector(`[data-column="${this.sortBy}"] .sort-icon`);
        if (activeIcon) {
            activeIcon.style.opacity = '1';
            activeIcon.style.color = '#667eea';
        }
    }

    showDeleteConfirmation(id, name) {
        this.deleteItemId = id;
        this.deleteItemName = name;
        
        const modal = document.getElementById('deleteModal');
        const itemNameSpan = document.getElementById('deleteItemName');
        
        if (modal && itemNameSpan) {
            itemNameSpan.textContent = name;
            modal.style.display = 'flex';
        }
    }

    closeDeleteModal() {
        const modal = document.getElementById('deleteModal');
        if (modal) {
            modal.style.display = 'none';
        }
        this.deleteItemId = null;
        this.deleteItemName = '';
    }

    async deleteRandevu() {
        if (!this.deleteItemId) return;

        const confirmBtn = document.getElementById('confirmDeleteBtn');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');
        
        // Show loading state
        confirmBtn.disabled = true;
        btnText.style.display = 'none';
        btnLoader.style.display = 'block';

        try {
            const response = await fetch(`/backend/randevu/${this.deleteItemId}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            const result = await response.json();

            if (result.status === 200) {
                this.showAlert('success', 'Randevu başarıyla silindi.');
                setTimeout(() => {
                    window.location.reload();
                }, 1500);
            } else {
                this.showAlert('error', result.message || 'Randevu silinirken bir hata oluştu.');
            }
        } catch (error) {
            console.error('Delete error:', error);
            this.showAlert('error', 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
        } finally {
            // Hide loading state
            confirmBtn.disabled = false;
            btnText.style.display = 'block';
            btnLoader.style.display = 'none';
            this.closeDeleteModal();
        }
    }

    showAlert(type, message) {
        const alertId = type === 'success' ? 'successMessage' : 'errorMessage';
        const alert = document.getElementById(alertId);
        const alertText = alert.querySelector('.alert-text');
        
        if (alert && alertText) {
            alertText.textContent = message;
            alert.classList.add('show');
            
            setTimeout(() => {
                alert.classList.remove('show');
            }, 5000);
        }
    }

    closeAlert(alertId) {
        const alert = document.getElementById(alertId);
        if (alert) {
            alert.classList.remove('show');
        }
    }

    setupMobileOptimizations() {
        // Handle window resize
        window.addEventListener('resize', this.debounce(() => {
            this.handleResize();
        }, 250));

        // Initial resize check
        this.handleResize();
    }

    handleResize() {
        const isMobile = window.innerWidth <= 768;
        
        if (isMobile && this.currentView === 'table') {
            this.toggleView('grid');
        }
    }

    bindRefreshButton() {
        const refreshBtn = document.getElementById('refreshBtn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                window.location.reload();
            });
        }
    }

    setupExport() {
        const exportBtn = document.getElementById('exportBtn');
        if (exportBtn) {
            exportBtn.addEventListener('click', () => {
                this.exportData();
            });
        }
    }

    exportData() {
        // Create CSV content
        const headers = [
            'Hasta Adı',
            'Hasta Soyadı',
            'Telefon',
            'E-posta',
            'Randevu Tarihi',
            'Randevu Saati',
            'Doktor',
            'Şube',
            'Durum',
            'Ödeme Durumu',
            'Ücret',
            'Oluşturma Tarihi'
        ];

        const csvContent = [
            headers.join(','),
            ...this.randevular.map(randevu => [
                `"${randevu.patient_first_name || ''}"`,
                `"${randevu.patient_last_name || ''}"`,
                `"${randevu.patient_phone || ''}"`,
                `"${randevu.patient_email || ''}"`,
                `"${randevu.appointment_date || ''}"`,
                `"${randevu.appointment_time || ''}"`,
                `"${randevu.doctor_name || ''}"`,
                `"${randevu.sube_name || ''}"`,
                `"${randevu.status || ''}"`,
                `"${randevu.payment_status || ''}"`,
                `"${randevu.price || ''}"`,
                `"${randevu.created_at || ''}"`
            ].join(','))
        ].join('\n');

        // Create and download file
        const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
        const link = document.createElement('a');
        const url = URL.createObjectURL(blob);
        link.setAttribute('href', url);
        link.setAttribute('download', `randevular_${new Date().toISOString().split('T')[0]}.csv`);
        link.style.visibility = 'hidden';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    }
}

// Global functions for modal handling
function closeDeleteModal() {
    const manager = window.randevularManager;
    if (manager) {
        manager.closeDeleteModal();
    }
}

function closeAlert(alertId) {
    const manager = window.randevularManager;
    if (manager) {
        manager.closeAlert(alertId);
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.randevularManager = new RandevularListManager();
});
