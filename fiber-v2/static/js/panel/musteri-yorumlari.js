class TestimonialsListManager {
    constructor() {
        this.data = window.pageData || {};
        this.items = this.data.testimonials || [];
        this.currentPage = this.data.currentPage || 1;
        this.perPage = this.data.perPage || 10;
        this.totalCount = this.data.count || 0;
        this.currentSort = { column: 'created_at', order: 'desc' };
        this.currentFilters = { search: '', status: '' };
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
        this.bindRefreshButton();
        this.bindImageFallbacks();
    }

    setupEventListeners() {
        document.addEventListener('click', (e) => {
            if (!e.target.closest('.actions-wrapper')) {
                this.closeAllDropdowns();
            }
            
            // Close modal when clicking outside
            if (e.target.classList.contains('modal')) {
                this.closeDeleteModal();
            }
        });

        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.closeAllDropdowns();
                const modal = document.querySelector('.modal.show');
                if (modal) this.closeDeleteModal();
            }
        });
        window.addEventListener('resize', () => this.closeAllDropdowns());
    }

    setupDropdowns() {
        const actionTriggers = document.querySelectorAll('.action-trigger');
        actionTriggers.forEach(trigger => {
            trigger.addEventListener('click', (e) => {
                e.stopPropagation();
                const dropdown = trigger.parentNode.querySelector('.action-dropdown');
                const isOpen = dropdown.classList.contains('show');
                this.closeAllDropdowns();
                if (!isOpen) dropdown.classList.add('show');
            });
        });

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

    closeAllDropdowns() {
        document.querySelectorAll('.action-dropdown.show').forEach(d => d.classList.remove('show'));
    }

    setupSearch() {
        const searchInput = document.getElementById('searchInput');
        const searchClear = document.getElementById('searchClear');
        if (!searchInput) return;
        let t;
        searchInput.addEventListener('input', (e) => {
            clearTimeout(t);
            const value = e.target.value.trim();
            searchClear.style.display = value ? 'block' : 'none';
            t = setTimeout(() => {
                this.currentFilters.search = value;
                this.currentPage = 1;
                this.filterAndReload();
            }, 300);
        });
        searchClear.addEventListener('click', () => {
            searchInput.value = '';
            searchClear.style.display = 'none';
            this.currentFilters.search = '';
            this.currentPage = 1;
            this.filterAndReload();
        });
    }

    setupFilters() {
        const statusFilter = document.getElementById('statusFilter');
        const sortBy = document.getElementById('sortBy');
        const sortOrder = document.getElementById('sortOrder');
        if (statusFilter) statusFilter.addEventListener('change', (e) => { this.currentFilters.status = e.target.value; this.currentPage = 1; this.filterAndReload(); });
        if (sortBy) sortBy.addEventListener('change', (e) => { this.currentSort.column = e.target.value; this.filterAndReload(); });
        if (sortOrder) sortOrder.addEventListener('change', (e) => { this.currentSort.order = e.target.value; this.filterAndReload(); });
    }

    setupTableSorting() {
        const sortableHeaders = document.querySelectorAll('.data-table th.sortable');
        sortableHeaders.forEach(header => {
            header.addEventListener('click', () => {
                const column = header.getAttribute('data-column');
                if (this.currentSort.column === column) {
                    this.currentSort.order = this.currentSort.order === 'asc' ? 'desc' : 'asc';
                } else {
                    this.currentSort.column = column;
                    this.currentSort.order = 'asc';
                }
                this.updateSortHeaders();
                this.filterAndReload();
            });
        });
    }

    updateSortHeaders() {
        const headers = document.querySelectorAll('.data-table th.sortable');
        headers.forEach(header => {
            header.classList.remove('asc', 'desc');
            if (header.getAttribute('data-column') === this.currentSort.column) {
                header.classList.add(this.currentSort.order);
            }
        });
    }

    setupViewToggle() {
        const viewButtons = document.querySelectorAll('.view-btn');
        const tableView = document.getElementById('tableView');
        const gridView = document.getElementById('gridView');
        viewButtons.forEach(button => {
            button.addEventListener('click', () => {
                const view = button.getAttribute('data-view');
                viewButtons.forEach(btn => btn.classList.remove('active'));
                button.classList.add('active');
                if (view === 'grid') {
                    tableView.style.display = 'none';
                    gridView.style.display = 'block';
                    this.currentView = 'grid';
                } else {
                    tableView.style.display = 'block';
                    gridView.style.display = 'none';
                    this.currentView = 'table';
                }
                setTimeout(() => this.setupDropdowns(), 100);
            });
        });
    }

    setupPagination() {
        this.renderPagination();
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

    renderPagination() {
        const paginationContainer = document.getElementById('pagination');
        if (!paginationContainer || this.totalCount === 0) return;
        const totalPages = Math.ceil(this.totalCount / this.perPage);
        const currentPage = this.currentPage;
        let html = '';
        html += `<button class="pagination-btn ${currentPage === 1 ? 'disabled' : ''}" data-page="${currentPage - 1}" ${currentPage === 1 ? 'disabled' : ''}><i class="fas fa-chevron-left"></i></button>`;
        const startPage = Math.max(1, currentPage - 2);
        const endPage = Math.min(totalPages, currentPage + 2);
        if (startPage > 1) {
            html += `<button class="pagination-btn" data-page="1">1</button>`;
            if (startPage > 2) html += `<span class="pagination-ellipsis">...</span>`;
        }
        for (let i = startPage; i <= endPage; i++) {
            html += `<button class="pagination-btn ${i === currentPage ? 'active' : ''}" data-page="${i}">${i}</button>`;
        }
        if (endPage < totalPages) {
            if (endPage < totalPages - 1) html += `<span class="pagination-ellipsis">...</span>`;
            html += `<button class="pagination-btn" data-page="${totalPages}">${totalPages}</button>`;
        }
        html += `<button class="pagination-btn ${currentPage === totalPages ? 'disabled' : ''}" data-page="${currentPage + 1}" ${currentPage === totalPages ? 'disabled' : ''}><i class="fas fa-chevron-right"></i></button>`;
        paginationContainer.innerHTML = html;
        paginationContainer.querySelectorAll('.pagination-btn:not(.disabled)').forEach(btn => {
            btn.addEventListener('click', () => {
                const page = parseInt(btn.getAttribute('data-page'));
                if (page && page !== this.currentPage) {
                    this.currentPage = page;
                    this.filterAndReload();
                }
            });
        });
    }

    async filterAndReload() {
        try {
            const params = new URLSearchParams();
            params.set('page', this.currentPage);
            params.set('per_page', this.perPage);
            params.set('sort_by', this.currentSort.column);
            params.set('sort_order', this.currentSort.order);
            if (this.currentFilters.search) params.set('search', this.currentFilters.search);
            if (this.currentFilters.status) params.set('status', this.currentFilters.status);
            window.location.href = `/panel/musteri-yorumlari?${params.toString()}`;
        } catch (e) {
            console.error(e);
        }
    }

    showDeleteConfirmation(id, name) {
        // Create modal if it doesn't exist
        let modal = document.getElementById('deleteModal');
        if (!modal) {
            modal = this.createDeleteModal();
        }

        const nameSpan = modal.querySelector('#deleteItemName');
        const confirmBtn = modal.querySelector('#confirmDeleteBtn');

        if (nameSpan) nameSpan.textContent = name;
        
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);

        // Remove existing listeners
        const newConfirmBtn = confirmBtn.cloneNode(true);
        confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);

        // Add new listener
        newConfirmBtn.addEventListener('click', () => {
            this.deleteTestimonial(id);
        });

        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    createDeleteModal() {
        const modal = document.createElement('div');
        modal.id = 'deleteModal';
        modal.className = 'modal';
        modal.style.display = 'none';
        modal.innerHTML = `
            <div class="modal-content danger">
                <div class="modal-icon">
                    <i class="icon-alert-circle"></i>
                </div>
                <h3 class="modal-title">Müşteri Yorumunu Sil</h3>
                <p class="modal-message">
                    "<span id="deleteItemName"></span>" adlı müşteri yorumunu silmek istediğinizden emin misiniz? 
                    Bu işlem geri alınamaz ve tüm ilişkili veriler silinecektir.
                </p>
                <div class="modal-actions">
                    <button type="button" class="btn btn-secondary" onclick="closeDeleteModal()">İptal</button>
                    <button type="button" class="btn btn-danger" id="confirmDeleteBtn">
                        <i class="icon-trash-2"></i>
                        <span class="btn-text">Sil</span>
                        <div class="btn-loader" style="display: none;">
                            <div class="spinner"></div>
                        </div>
                    </button>
                </div>
            </div>
        `;
        document.body.appendChild(modal);
        return modal;
    }

    async deleteTestimonial(id) {
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

            const response = await fetch(`/backend/testimonial/${id}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const responseData = await response.json().catch(() => ({}));
            
            if (responseData.status === 201) {
                this.closeDeleteModal();
                this.showAlert('Müşteri yorumu başarıyla silindi.', 'success');
                
                // Reload page after 2 seconds
                setTimeout(() => {
                    window.location.reload();
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

    closeAlert(alertId) {
        const alert = document.getElementById(alertId);
        if (!alert) return;

        alert.classList.remove('show');
        setTimeout(() => {
            alert.style.display = 'none';
        }, 300);
    }

    closeModal(id) {
        const modal = document.getElementById(id);
        if (!modal) return;
        modal.classList.remove('show');
        setTimeout(() => modal.style.display = 'none', 300);
    }

    bindRefreshButton() {
        const refreshBtn = document.getElementById('refreshBtn');
        if (refreshBtn) refreshBtn.addEventListener('click', () => window.location.reload());
    }

    bindImageFallbacks() {
        const avatars = document.querySelectorAll('.user-avatar img');
        avatars.forEach(img => {
            img.addEventListener('error', () => {
                const parent = img.parentElement;
                if (!parent) return;
                img.remove();
                const icon = document.createElement('i');
                icon.className = 'fas fa-user';
                parent.appendChild(icon);
            });
        });
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

document.addEventListener('DOMContentLoaded', () => {
    new TestimonialsListManager();
});

window.TestimonialsListManager = TestimonialsListManager;

