/**
 * Brans (Individual Branch View) Page JavaScript
 * Handles delete functionality, image zoom, change head doctor, and interactive elements
 * Adapted from sube.js for branch management
 */

class BransViewManager {
    constructor() {
        this.bransData = window.bransData || {};
        this.currentZoomedImage = null;
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupImageZoom();
        this.setupDeleteButton();
        this.setupDropdown();
        this.setupAddDoctorButton();
        this.setupRemoveDoctorButton();
        this.setupChangeHeadDoctorButton();
        this.formatDates();
        this.setupCopyToClipboard();
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
                const addDoctorModal = document.getElementById('addDoctorModal');
                const removeDoctorModal = document.getElementById('removeDoctorModal');
                const changeHeadDoctorModal = document.getElementById('changeHeadDoctorModal');
                const imageModal = document.getElementById('imageZoomModal');
                
                if (deleteModal && deleteModal.classList.contains('show')) {
                    this.closeDeleteModal();
                } else if (addDoctorModal && addDoctorModal.classList.contains('show')) {
                    this.closeAddDoctorModal();
                } else if (removeDoctorModal && removeDoctorModal.classList.contains('show')) {
                    this.closeRemoveDoctorModal();
                } else if (changeHeadDoctorModal && changeHeadDoctorModal.classList.contains('show')) {
                    this.closeChangeHeadDoctorModal();
                } else if (imageModal && imageModal.classList.contains('show')) {
                    this.closeImageModal();
                } else {
                    // Close dropdown if no modal is open
                    this.closeDropdown();
                }
            }
        });

        // Close modals when clicking outside
        document.addEventListener('click', (e) => {
            const deleteModal = document.getElementById('deleteModal');
            const addDoctorModal = document.getElementById('addDoctorModal');
            const removeDoctorModal = document.getElementById('removeDoctorModal');
            const changeHeadDoctorModal = document.getElementById('changeHeadDoctorModal');
            const imageModal = document.getElementById('imageZoomModal');
            
            if (e.target === deleteModal) {
                this.closeDeleteModal();
            } else if (e.target === addDoctorModal) {
                this.closeAddDoctorModal();
            } else if (e.target === removeDoctorModal) {
                this.closeRemoveDoctorModal();
            } else if (e.target === changeHeadDoctorModal) {
                this.closeChangeHeadDoctorModal();
            } else if (e.target === imageModal) {
                this.closeImageModal();
            }
        });

        // Handle window resize
        window.addEventListener('resize', () => {
            this.handleResize();
        });
    }

    /**
     * Setup image zoom functionality
     */
    setupImageZoom() {
        const zoomableImages = document.querySelectorAll('.zoomable');
        
        zoomableImages.forEach(image => {
            image.addEventListener('click', (e) => {
                e.preventDefault();
                this.openImageModal(image);
            });

            // Add loading state
            image.addEventListener('load', () => {
                image.classList.add('loaded');
            });

            // Add error handling
            image.addEventListener('error', () => {
                image.classList.add('error');
                const parent = image.closest('.image-preview');
                if (parent) {
                    parent.innerHTML = `
                        <div class="image-error">
                            <i class="icon-image-off"></i>
                            <span>Görsel yüklenemedi</span>
                        </div>
                    `;
                }
            });
        });
    }

    /**
     * Open image in zoom modal
     */
    openImageModal(imageElement) {
        const modal = document.getElementById('imageZoomModal');
        const zoomedImage = document.getElementById('zoomedImage');
        const imageTitle = document.getElementById('imageTitle');
        
        if (!modal || !zoomedImage) return;

        // Set image source and title
        zoomedImage.src = imageElement.src;
        zoomedImage.alt = imageElement.alt;
        this.currentZoomedImage = imageElement.src;
        
        // Set title based on alt text or default
        const title = imageElement.alt || 'Bölüm Görseli';
        if (imageTitle) imageTitle.textContent = title;
        
        // Show modal
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
        
        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Close image modal
     */
    closeImageModal() {
        const modal = document.getElementById('imageZoomModal');
        if (!modal) return;

        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
            this.currentZoomedImage = null;
        }, 300);
        
        // Restore body scroll
        document.body.style.overflow = '';
    }

    /**
     * Download current zoomed image
     */
    downloadImage() {
        if (!this.currentZoomedImage) return;

        const link = document.createElement('a');
        link.href = this.currentZoomedImage;
        link.download = this.currentZoomedImage.split('/').pop() || 'brans-gorsel';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        
        this.showAlert('Görsel indiriliyor...', 'success');
    }

    /**
     * Open image in new tab
     */
    openImageInNewTab() {
        if (!this.currentZoomedImage) return;

        window.open(this.currentZoomedImage, '_blank');
    }

    /**
     * Setup delete button functionality
     */
    setupDeleteButton() {
        const deleteBtn = document.getElementById('deleteBransBtn');
        
        if (!deleteBtn) return;

        deleteBtn.addEventListener('click', () => {
            const id = deleteBtn.getAttribute('data-id');
            const name = deleteBtn.getAttribute('data-name');
            this.showDeleteConfirmation(id, name);
        });
    }

    /**
     * Setup dropdown functionality
     */
    setupDropdown() {
        // Wait a bit to ensure DOM is ready
        setTimeout(() => {
            const dropdownToggle = document.getElementById('doctorActionsDropdown');
            const dropdownMenu = dropdownToggle?.nextElementSibling;
            
            if (!dropdownToggle) {
                return;
            }
            
            if (!dropdownMenu) {
                return;
            }

            // Add click handler
            dropdownToggle.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                
                // Simple toggle
                if (dropdownMenu.classList.contains('show')) {
                    dropdownMenu.classList.remove('show');
                    dropdownToggle.setAttribute('aria-expanded', 'false');
                } else {
                    // Close any other open dropdowns first
                    document.querySelectorAll('.dropdown-menu.show').forEach(menu => {
                        menu.classList.remove('show');
                    });
                    
                    dropdownMenu.classList.add('show');
                    dropdownToggle.setAttribute('aria-expanded', 'true');
                }
            });

            // Close dropdown when clicking outside
            document.addEventListener('click', (e) => {
                if (!dropdownToggle.contains(e.target) && !dropdownMenu.contains(e.target)) {
                    dropdownMenu.classList.remove('show');
                    dropdownToggle.setAttribute('aria-expanded', 'false');
                }
            });
        }, 100);
    }

    /**
     * Setup add doctor button functionality
     */
    setupAddDoctorButton() {
        const addDoctorBtn = document.getElementById('addDoctorBtn');
        
        if (!addDoctorBtn) return;

        addDoctorBtn.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            this.showAddDoctorModal();
        });
    }

    /**
     * Setup remove doctor button functionality
     */
    setupRemoveDoctorButton() {
        const removeDoctorBtn = document.getElementById('removeDoctorBtn');
        
        if (!removeDoctorBtn) return;

        removeDoctorBtn.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            this.showRemoveDoctorModal();
        });
    }

    /**
     * Setup change head doctor button functionality
     */
    setupChangeHeadDoctorButton() {
        const changeHeadDoctorBtn = document.getElementById('changeHeadDoctorBtn');
        
        if (!changeHeadDoctorBtn) return;

        changeHeadDoctorBtn.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            this.showChangeHeadDoctorModal();
        });
    }

    /**
     * Show add doctor modal
     */
    showAddDoctorModal() {
        const modal = document.getElementById('addDoctorModal');
        const confirmBtn = document.getElementById('confirmAddDoctorBtn');

        if (!modal || !confirmBtn) return;

        // Close dropdown
        this.closeDropdown();

        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);

        // Load available doctors
        this.loadAvailableDoctors();

        // Remove existing listeners
        const newConfirmBtn = confirmBtn.cloneNode(true);
        confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);

        // Add new listener
        newConfirmBtn.addEventListener('click', async () => {
            await this.addDoctor();
        });

        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Show remove doctor modal
     */
    showRemoveDoctorModal() {
        const modal = document.getElementById('removeDoctorModal');
        const confirmBtn = document.getElementById('confirmRemoveDoctorBtn');

        if (!modal || !confirmBtn) return;

        // Close dropdown
        this.closeDropdown();

        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);

        // Remove existing listeners
        const newConfirmBtn = confirmBtn.cloneNode(true);
        confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);

        // Add new listener
        newConfirmBtn.addEventListener('click', async () => {
            await this.removeDoctor();
        });

        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Show change head doctor modal
     */
    showChangeHeadDoctorModal() {
        const modal = document.getElementById('changeHeadDoctorModal');
        const confirmBtn = document.getElementById('confirmChangeHeadDoctorBtn');

        if (!modal || !confirmBtn) return;

        // Close dropdown
        this.closeDropdown();

        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);

        // Remove existing listeners
        const newConfirmBtn = confirmBtn.cloneNode(true);
        confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);

        // Add new listener
        newConfirmBtn.addEventListener('click', () => {
            this.changeHeadDoctor();
        });

        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Close dropdown
     */
    closeDropdown() {
        const dropdownToggle = document.getElementById('doctorActionsDropdown');
        const dropdownMenu = dropdownToggle?.nextElementSibling;
        
        if (dropdownMenu) {
            dropdownMenu.classList.remove('show');
        }
        if (dropdownToggle) {
            dropdownToggle.setAttribute('aria-expanded', 'false');
        }
    }

    /**
     * Close add doctor modal
     */
    closeAddDoctorModal() {
        const modal = document.getElementById('addDoctorModal');
        if (!modal) return;

        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
        
        // Restore body scroll
        document.body.style.overflow = '';
    }

    /**
     * Close remove doctor modal
     */
    closeRemoveDoctorModal() {
        const modal = document.getElementById('removeDoctorModal');
        if (!modal) return;

        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
        
        // Restore body scroll
        document.body.style.overflow = '';
    }

    /**
     * Close change head doctor modal
     */
    closeChangeHeadDoctorModal() {
        const modal = document.getElementById('changeHeadDoctorModal');
        if (!modal) return;

        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
        
        // Restore body scroll
        document.body.style.overflow = '';
    }

    /**
     * Change head doctor
     */
    async changeHeadDoctor() {
        const confirmBtn = document.getElementById('confirmChangeHeadDoctorBtn');
        const newHeadDoctorSelect = document.getElementById('newHeadDoctor');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');

        if (!confirmBtn || !newHeadDoctorSelect || !btnText || !btnLoader) return;

        const newHeadDoctorId = newHeadDoctorSelect.value;
        if (!newHeadDoctorId) {
            this.showAlert('Lütfen bir başhekim seçin.', 'error');
            return;
        }

        try {
            // Show loading state
            confirmBtn.disabled = true;
            btnText.style.opacity = '0';
            btnLoader.style.display = 'block';

            const response = await fetch(`/backend/branch/${this.bransData.id}/change-head-doctor`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                },
                body: JSON.stringify({
                    head_drid: newHeadDoctorId
                })
            });

            const responseData = await response.json().catch(() => ({}));
            
            if (responseData.status === 201) {
                this.closeChangeHeadDoctorModal();
                this.showAlert('Başhekim başarıyla değiştirildi.', 'success');
                
                // Reload page to reflect changes
                setTimeout(() => {
                    window.location.reload();
                }, 1500);

            } else {
                this.showAlert(responseData.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            }
        } catch (error) {
            console.error('Change head doctor error:', error);
            this.showAlert('Bağlantı hatası oluştu.', 'error');
        } finally {
            // Reset button state
            confirmBtn.disabled = false;
            btnText.style.opacity = '1';
            btnLoader.style.display = 'none';
        }
    }

    /**
     * Load available doctors for adding to branch
     */
    async loadAvailableDoctors() {
        const select = document.getElementById('doctorToAdd');
        if (!select) return;

        try {
            const response = await fetch(`/backend/branch/${this.bransData.id}/get-doctors-for-adding-branch`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const result = await response.json();

            if (result.status === 200) {
                // Clear existing options except the first one
                select.innerHTML = '<option value="">Doktor Seçiniz</option>';
                
                // Add available doctors
                result.data.forEach(doctor => {
                    const option = document.createElement('option');
                    option.value = doctor.drid;
                    option.textContent = `${doctor.title} ${doctor.first_name} ${doctor.last_name}`;
                    select.appendChild(option);
                });
            } else {
                this.showAlert('Doktorlar yüklenirken hata oluştu.', 'error');
            }
        } catch (error) {
            console.error('Load doctors error:', error);
            this.showAlert('Bağlantı hatası oluştu.', 'error');
        }
    }

    /**
     * Add doctor to branch
     */
    async addDoctor() {
        const confirmBtn = document.getElementById('confirmAddDoctorBtn');
        const doctorSelect = document.getElementById('doctorToAdd');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');

        if (!confirmBtn || !doctorSelect || !btnText || !btnLoader) return;

        const doctorId = doctorSelect.value;
        if (!doctorId) {
            this.showAlert('Lütfen bir doktor seçin.', 'error');
            return;
        }

        try {
            // Show loading state
            confirmBtn.disabled = true;
            btnText.style.opacity = '0';
            btnLoader.style.display = 'block';

            const response = await fetch(`/backend/doctor/${doctorId}/add-to-a-branch`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                },
                body: JSON.stringify({
                    brid: this.bransData.id.toString()
                })
            });

            const result = await response.json();

            if (result.status === 201) {
                this.showAlert('Doktor başarıyla bölüme eklendi.', 'success');
                this.closeAddDoctorModal();
                // Reload page to show updated data
                setTimeout(() => {
                    window.location.reload();
                }, 1500);
            } else {
                this.showAlert(result.message || 'Doktor eklenirken hata oluştu.', 'error');
            }
        } catch (error) {
            console.error('Add doctor error:', error);
            this.showAlert('Bağlantı hatası oluştu.', 'error');
        } finally {
            // Reset button state
            confirmBtn.disabled = false;
            btnText.style.opacity = '1';
            btnLoader.style.display = 'none';
        }
    }

    /**
     * Remove doctor from branch
     */
    async removeDoctor() {
        const confirmBtn = document.getElementById('confirmRemoveDoctorBtn');
        const doctorSelect = document.getElementById('doctorToRemove');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');

        if (!confirmBtn || !doctorSelect || !btnText || !btnLoader) return;

        const doctorId = doctorSelect.value;
        if (!doctorId) {
            this.showAlert('Lütfen bir doktor seçin.', 'error');
            return;
        }

        try {
            // Show loading state
            confirmBtn.disabled = true;
            btnText.style.opacity = '0';
            btnLoader.style.display = 'block';

            const response = await fetch(`/backend/doctor/${doctorId}/remove-from-a-branch`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const result = await response.json();

            if (result.status === 201) {
                this.showAlert('Doktor başarıyla bölümden çıkarıldı.', 'success');
                this.closeRemoveDoctorModal();
                // Reload page to show updated data
                setTimeout(() => {
                    window.location.reload();
                }, 1500);
            } else {
                this.showAlert(result.message || 'Doktor çıkarılırken hata oluştu.', 'error');
            }
        } catch (error) {
            console.error('Remove doctor error:', error);
            this.showAlert('Bağlantı hatası oluştu.', 'error');
        } finally {
            // Reset button state
            confirmBtn.disabled = false;
            btnText.style.opacity = '1';
            btnLoader.style.display = 'none';
        }
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
            this.deleteBrans(id);
        });

        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Delete branch
     */
    async deleteBrans(id) {
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');

        if (!confirmBtn || !btnText || !btnLoader) return;

        try {
            // Show loading state
            confirmBtn.disabled = true;
            btnText.style.opacity = '0';
            btnLoader.style.display = 'block';

            const response = await fetch(`/backend/branch/${id}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const responseData = await response.json().catch(() => ({}));
            
            if (responseData.status === 201) {
                this.closeDeleteModal();
                this.showAlert('Bölüm başarıyla silindi.', 'success');
                
                // Redirect to branches list after 2 seconds
                setTimeout(() => {
                    window.location.href = '/panel/branslar';
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
     * Format dates for better display
     */
    formatDates() {
        const dateElements = document.querySelectorAll('[data-date]');
        
        dateElements.forEach(element => {
            const dateValue = element.getAttribute('data-date');
            if (dateValue) {
                try {
                    const date = new Date(dateValue);
                    const formattedDate = date.toLocaleDateString('tr-TR', {
                        year: 'numeric',
                        month: '2-digit',
                        day: '2-digit',
                        hour: '2-digit',
                        minute: '2-digit'
                    });
                    element.textContent = formattedDate;
                } catch (error) {
                    console.warn('Date formatting error:', error);
                }
            }
        });
    }

    /**
     * Setup copy to clipboard functionality
     */
    setupCopyToClipboard() {
        // Add copy buttons to contact links
        const contactLinks = document.querySelectorAll('.contact-link');
        contactLinks.forEach(link => {
            const copyBtn = document.createElement('button');
            copyBtn.className = 'btn-icon copy-btn';
            copyBtn.innerHTML = '<i class="icon-copy"></i>';
            copyBtn.title = 'Kopyala';
            copyBtn.style.marginLeft = '8px';
            
            copyBtn.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                const text = link.textContent.trim();
                this.copyToClipboard(text, 'İletişim bilgisi kopyalandı!');
            });
            
            link.parentNode.appendChild(copyBtn);
        });

        // Add copy functionality to address
        const addressElements = document.querySelectorAll('.info-value');
        addressElements.forEach(element => {
            const label = element.parentNode.querySelector('.info-label');
            if (label && label.textContent.includes('Adres')) {
                element.style.cursor = 'pointer';
                element.title = 'Kopyalamak için tıklayın';
                
                element.addEventListener('click', () => {
                    this.copyToClipboard(element.textContent, 'Adres kopyalandı!');
                });
            }
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
            const imageModal = document.getElementById('imageZoomModal');
            if (imageModal?.classList.contains('show')) {
                this.closeImageModal();
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
            
            // Ctrl/Cmd + D for delete
            if ((e.ctrlKey || e.metaKey) && e.key === 'd') {
                e.preventDefault();
                const deleteBtn = document.getElementById('deleteBransBtn');
                if (deleteBtn) {
                    deleteBtn.click();
                }
            }
            
            // Ctrl/Cmd + B for back to list
            if ((e.ctrlKey || e.metaKey) && e.key === 'b') {
                e.preventDefault();
                window.location.href = '/panel/branslar';
            }
        });
    }

    /**
     * Setup tooltips for truncated text
     */
    setupTooltips() {
        const truncatedElements = document.querySelectorAll('.truncate');
        
        truncatedElements.forEach(element => {
            if (element.scrollWidth > element.clientWidth) {
                element.title = element.textContent;
                element.style.cursor = 'help';
            }
        });
    }

    /**
     * Format bytes for display
     */
    formatBytes(bytes) {
        if (bytes === 0) return '0 Bytes';
        
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    /**
     * Animate elements on scroll
     */
    setupScrollAnimations() {
        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    entry.target.classList.add('animate-in');
                }
            });
        }, {
            threshold: 0.1,
            rootMargin: '0px 0px -50px 0px'
        });

        const cards = document.querySelectorAll('.content-card');
        cards.forEach(card => {
            observer.observe(card);
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

window.closeChangeHeadDoctorModal = function() {
    const modal = document.getElementById('changeHeadDoctorModal');
    if (modal) {
        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
        document.body.style.overflow = '';
    }
};

window.closeImageModal = function() {
    const modal = document.getElementById('imageZoomModal');
    if (modal) {
        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
        document.body.style.overflow = '';
    }
};

window.downloadImage = function() {
    const manager = window.bransViewManager;
    if (manager) {
        manager.downloadImage();
    }
};

window.openImageInNewTab = function() {
    const manager = window.bransViewManager;
    if (manager) {
        manager.openImageInNewTab();
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

function formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.bransViewManager = new BransViewManager();
    
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
        
        .image-error {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            padding: 20px;
            background: var(--bg-tertiary);
            border: 1px dashed var(--border-color);
            border-radius: var(--border-radius);
            color: var(--text-muted);
            font-size: 14px;
            gap: 8px;
        }
        
        .image-error i {
            font-size: 24px;
        }
        
        .copy-btn {
            opacity: 0;
            transition: opacity 0.2s ease;
            border: 1px solid var(--border-color);
            background: var(--bg-tertiary);
            border-radius: var(--border-radius);
            padding: 4px 8px;
            cursor: pointer;
            color: var(--text-secondary);
        }
        
        .copy-btn:hover {
            background: var(--primary-color);
            color: var(--text-white);
            border-color: var(--primary-color);
        }
        
        .info-item:hover .copy-btn {
            opacity: 1;
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

// Global functions for modal closing (called from HTML onclick)
function toggleDoctorDropdown() {
    const dropdownToggle = document.getElementById('doctorActionsDropdown');
    const dropdownMenu = dropdownToggle?.nextElementSibling;
    
    if (!dropdownToggle || !dropdownMenu) {
        return;
    }
    
    if (dropdownMenu.style.display === 'block' || dropdownMenu.classList.contains('show')) {
        dropdownMenu.style.display = 'none';
        dropdownMenu.classList.remove('show');
        dropdownToggle.setAttribute('aria-expanded', 'false');
    } else {
        // Close any other open dropdowns first
        document.querySelectorAll('.dropdown-menu').forEach(menu => {
            menu.style.display = 'none';
            menu.classList.remove('show');
        });
        
        dropdownMenu.style.display = 'block';
        dropdownMenu.style.opacity = '1';
        dropdownMenu.style.visibility = 'visible';
        dropdownMenu.style.transform = 'translateY(0)';
        dropdownMenu.classList.add('show');
        dropdownToggle.setAttribute('aria-expanded', 'true');
    }
}

function closeAddDoctorModal() {
    if (window.bransViewManager) {
        window.bransViewManager.closeAddDoctorModal();
    }
}

function closeRemoveDoctorModal() {
    if (window.bransViewManager) {
        window.bransViewManager.closeRemoveDoctorModal();
    }
}

function closeChangeHeadDoctorModal() {
    if (window.bransViewManager) {
        window.bransViewManager.closeChangeHeadDoctorModal();
    }
}

function closeDeleteModal() {
    if (window.bransViewManager) {
        window.bransViewManager.closeDeleteModal();
    }
}

function closeImageModal() {
    if (window.bransViewManager) {
        window.bransViewManager.closeImageModal();
    }
}

function closeAlert(alertId) {
    if (window.bransViewManager) {
        window.bransViewManager.closeAlert(alertId);
    }
}


// Export for potential external use
window.BransViewManager = BransViewManager;
