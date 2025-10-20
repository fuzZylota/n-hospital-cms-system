/**
 * Doktor (Individual Doctor View) Page JavaScript
 * Handles delete functionality, image zoom, modal interactions, and AJAX operations
 */

class DoktorViewManager {
    constructor() {
        this.doktorData = window.doktorData || {};
        this.currentZoomedImage = null;
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupImageZoom();
        this.setupDeleteButton();
        this.setupDropdown();
        this.setupExpertiseModal();
        this.setupExperienceModal();
        this.setupExpertiseDeleteButtons();
        this.setupExperienceDeleteButtons();
        this.setupEditExperienceHandlers();
        this.formatDates();
        this.setupCopyToClipboard();
        this.setupKeyboardShortcuts();
    }

    /**
     * Convert date for backend submission (ISO or Go zero time)
     */
    convertDateForBackend(dateValue) {
        if (!dateValue || dateValue === '') {
            return '0001-01-01T00:00:00Z';
        }

        if (/^\d{4}-\d{2}-\d{2}$/.test(dateValue)) {
            return dateValue + 'T00:00:00Z';
        }

        if (String(dateValue).includes('00:00, 01/01/0001')) {
            return '0001-01-01T00:00:00Z';
        }

        const date = new Date(dateValue);
        if (!isNaN(date.getTime())) {
            return date.toISOString();
        }

        return '0001-01-01T00:00:00Z';
    }

    /**
     * Setup all event listeners
     */
    setupEventListeners() {
        // Handle escape key for modals
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                const deleteModal = document.getElementById('deleteModal');
                const imageModal = document.getElementById('imageZoomModal');
                const expertiseModal = document.getElementById('addExpertiseModal');
                const experienceModal = document.getElementById('addExperienceModal');
                const editExperienceModal = document.getElementById('editExperienceModal');

                if (deleteModal && deleteModal.style.display !== 'none') {
                    this.closeDeleteModal();
                } else if (imageModal && imageModal.style.display !== 'none') {
                    this.closeImageModal();
                } else if (expertiseModal && expertiseModal.style.display !== 'none') {
                    this.closeAddExpertiseModal();
                } else if (experienceModal && experienceModal.style.display !== 'none') {
                    this.closeAddExperienceModal();
                } else if (editExperienceModal && editExperienceModal.style.display !== 'none') {
                    this.closeEditExperienceModal();
                } else {
                    this.closeDropdown();
                }
            }
        });

        // Close modals when clicking outside
        document.addEventListener('click', (e) => {
            const deleteModal = document.getElementById('deleteModal');
            const imageModal = document.getElementById('imageZoomModal');
            const expertiseModal = document.getElementById('addExpertiseModal');
            const experienceModal = document.getElementById('addExperienceModal');
            const editExperienceModal = document.getElementById('editExperienceModal');

            if (e.target === deleteModal) {
                this.closeDeleteModal();
            } else if (e.target === imageModal) {
                this.closeImageModal();
            } else if (e.target === expertiseModal) {
                this.closeAddExpertiseModal();
            } else if (e.target === experienceModal) {
                this.closeAddExperienceModal();
            } else if (e.target === editExperienceModal) {
                this.closeEditExperienceModal();
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
                const parent = image.closest('.doctor-photo');
                if (parent) {
                    parent.innerHTML = `
                        <div class="image-error">
                            <i class="fas fa-user-md"></i>
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
        
        this.currentZoomedImage = imageElement.src;
        zoomedImage.src = imageElement.src;
        zoomedImage.alt = imageElement.alt || 'Doktor Fotoğrafı';
        
        if (imageTitle) {
            imageTitle.textContent = imageElement.title || imageElement.alt || 'Doktor Fotoğrafı';
        }
        
        modal.style.display = 'flex';
        document.body.style.overflow = 'hidden';
        
        // Focus management for accessibility
        modal.focus();
    }

    /**
     * Close image zoom modal
     */
    closeImageModal() {
        const modal = document.getElementById('imageZoomModal');
        if (!modal) return;
        
        modal.style.display = 'none';
        document.body.style.overflow = '';
        this.currentZoomedImage = null;
    }

    /**
     * Setup delete button functionality
     */
    setupDeleteButton() {
        const deleteBtn = document.getElementById('deleteDoctorBtn');
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
                this.deleteDoctor();
            });
        }
    }

    /**
     * Setup dropdown functionality
     */
    setupDropdown() {
        const dropdownToggle = document.getElementById('doctorOptionsDropdown');
        const dropdownMenu = document.querySelector('#doctorOptionsDropdown + .dropdown-menu');
        
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
        const dropdownMenu = document.querySelector('#doctorOptionsDropdown + .dropdown-menu');
        const dropdownToggle = document.getElementById('doctorOptionsDropdown');
        
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
        const dropdownMenu = document.querySelector('#doctorOptionsDropdown + .dropdown-menu');
        const dropdownToggle = document.getElementById('doctorOptionsDropdown');
        
        if (!dropdownMenu || !dropdownToggle) return;
        
        dropdownMenu.classList.add('show');
        dropdownToggle.setAttribute('aria-expanded', 'true');
    }

    /**
     * Close dropdown menu
     */
    closeDropdown() {
        const dropdownMenu = document.querySelector('#doctorOptionsDropdown + .dropdown-menu');
        const dropdownToggle = document.getElementById('doctorOptionsDropdown');
        
        if (!dropdownMenu || !dropdownToggle) return;
        
        dropdownMenu.classList.remove('show');
        dropdownToggle.setAttribute('aria-expanded', 'false');
    }

    /**
     * Setup expertise modal functionality
     */
    setupExpertiseModal() {
        const addExpertiseBtn = document.getElementById('addExpertiseBtn');
        const saveExpertiseBtn = document.getElementById('saveExpertiseBtn');
        
        if (addExpertiseBtn) {
            addExpertiseBtn.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                this.showAddExpertiseModal();
            });
        }
        
        if (saveExpertiseBtn) {
            saveExpertiseBtn.addEventListener('click', (e) => {
                e.preventDefault();
                this.saveExpertise();
            });
        }
    }

    /**
     * Setup experience modal functionality
     */
    setupExperienceModal() {
        const addExperienceBtn = document.getElementById('addExperienceBtn');
        const saveExperienceBtn = document.getElementById('saveExperienceBtn');
        
        if (addExperienceBtn) {
            addExperienceBtn.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                this.showAddExperienceModal();
            });
        }
        
        if (saveExperienceBtn) {
            saveExperienceBtn.addEventListener('click', (e) => {
                e.preventDefault();
                this.saveExperience();
            });
        }

        // Initialize drag & drop for experience cover
        document.addEventListener('click', () => {
            // no-op to keep scope consistent
        });

        const area = document.getElementById('experienceCoverArea');
        if (area) {
            const input = area.querySelector('.file-input');
            const content = area.querySelector('.file-upload-content');
            const preview = area.querySelector('.file-preview');
            const previewImage = area.querySelector('.preview-image');
            const removeBtn = area.querySelector('.file-remove');
            const browseLink = area.querySelector('.file-browse');

            if (browseLink) {
                browseLink.addEventListener('click', (e) => {
                    e.preventDefault();
                    input && input.click();
                });
            }

            area.addEventListener('click', (e) => {
                if (e.target === area || e.target === content) {
                    input && input.click();
                }
            });

            area.addEventListener('dragover', (e) => {
                e.preventDefault();
                area.classList.add('dragover');
            });

            area.addEventListener('dragleave', (e) => {
                e.preventDefault();
                if (!area.contains(e.relatedTarget)) {
                    area.classList.remove('dragover');
                }
            });

            area.addEventListener('drop', (e) => {
                e.preventDefault();
                area.classList.remove('dragover');
                const files = e.dataTransfer.files;
                if (files && files.length > 0) {
                    this.handleExperienceCoverSelect(input, files[0], content, preview, previewImage, area);
                }
            });

            if (input) {
                input.addEventListener('change', (e) => {
                    if (e.target.files && e.target.files.length > 0) {
                        this.handleExperienceCoverSelect(input, e.target.files[0], content, preview, previewImage, area);
                    }
                });
            }

            if (removeBtn) {
                removeBtn.addEventListener('click', (e) => {
                    e.stopPropagation();
                    this.removeExperienceCover(input, content, preview, area);
                });
            }
        }
    }

    /**
     * Setup expertise delete buttons
     */
    setupExpertiseDeleteButtons() {
        const deleteButtons = document.querySelectorAll('.delete-expertise-btn');
        
        deleteButtons.forEach(button => {
            button.addEventListener('click', (e) => {
                e.preventDefault();
                const duid = button.getAttribute('data-duid');
                const name = button.getAttribute('data-name');
                this.deleteExpertise(duid, name);
            });
        });
    }

    /**
     * Setup experience delete buttons
     */
    setupExperienceDeleteButtons() {
        const deleteButtons = document.querySelectorAll('.delete-experience-btn');
        
        deleteButtons.forEach(button => {
            button.addEventListener('click', (e) => {
                e.preventDefault();
                const dtid = button.getAttribute('data-dtid');
                const name = button.getAttribute('data-name');
                this.deleteExperience(dtid, name);
            });
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
            nameSpan.textContent = this.doktorData.name || 'Bu doktor';
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
     * Show add expertise modal
     */
    showAddExpertiseModal() {
        const modal = document.getElementById('addExpertiseModal');
        if (!modal) return;
        
        this.closeDropdown();
        this.loadExpertiseOptions();
        modal.style.display = 'flex';
        document.body.style.overflow = 'hidden';
    }

    /**
     * Close add expertise modal
     */
    closeAddExpertiseModal() {
        const modal = document.getElementById('addExpertiseModal');
        if (!modal) return;
        
        modal.style.display = 'none';
        document.body.style.overflow = '';
        this.resetExpertiseForm();
    }

    /**
     * Show add experience modal
     */
    showAddExperienceModal() {
        const modal = document.getElementById('addExperienceModal');
        if (!modal) return;
        
        this.closeDropdown();
        modal.style.display = 'flex';
        document.body.style.overflow = 'hidden';
    }

    /**
     * Close add experience modal
     */
    closeAddExperienceModal() {
        const modal = document.getElementById('addExperienceModal');
        if (!modal) return;
        
        modal.style.display = 'none';
        document.body.style.overflow = '';
        this.resetExperienceForm();
    }

    /**
     * Load expertise options for select dropdown
     */
    async loadExpertiseOptions() {
        const select = document.getElementById('expertise_select');
        if (!select) return;
        
        try {
            const response = await fetch('/backend/doctor/get-all-expertises', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });
            
            const data = await response.json();
            
            if (data.status === 200) {
                select.innerHTML = '<option value="">Uzmanlık alanı seçin...</option>';
                
                data.data.forEach(expertise => {
                    const option = document.createElement('option');
                    option.value = expertise.uzid;
                    option.textContent = expertise.name;
                    select.appendChild(option);
                });
            }
        } catch (error) {
            console.error('Error loading expertise options:', error);
            this.showAlert('Uzmanlık alanları yüklenirken hata oluştu.', 'error');
        }
    }

    /**
     * Save new expertise
     */
    async saveExpertise() {
        const form = document.getElementById('addExpertiseForm');
        const saveBtn = document.getElementById('saveExpertiseBtn');
        
        if (!form || !saveBtn) return;
        
        const formData = new FormData(form);
        const data = {
            uzid: formData.get('uzid'),
            certification_date: formData.get('certification_date') || null,
            certification_institution: formData.get('certification_institution') || '',
            is_primary: formData.get('is_primary') === 'on'
        };
        
        if (!data.uzid) {
            this.showAlert('Lütfen bir uzmanlık alanı seçin.', 'error');
            return;
        }
        
        this.setButtonLoading(saveBtn, true);
        
        try {
            const response = await fetch(`/backend/doctor/${this.doktorData.id}/add-expertise`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data)
            });
            
            const result = await response.json();
            
            if (result.status === 201) {
                this.showAlert('Uzmanlık alanı başarıyla eklendi.', 'success');
                this.closeAddExpertiseModal();
                setTimeout(() => {
                    window.location.reload();
                }, 1500);
            } else {
                this.showAlert(result.message || 'Uzmanlık alanı eklenirken hata oluştu.', 'error');
            }
        } catch (error) {
            console.error('Error saving expertise:', error);
            this.showAlert('Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
        } finally {
            this.setButtonLoading(saveBtn, false);
        }
    }

    /**
     * Save new experience
     */
    async saveExperience() {
        const form = document.getElementById('addExperienceForm');
        const saveBtn = document.getElementById('saveExperienceBtn');
        
        if (!form || !saveBtn) return;
        
        const formData = new FormData(form);
        const data = {
            name: formData.get('name'),
            start_date: formData.get('start_date') || null,
            end_date: formData.get('end_date') || null,
            description: formData.get('description') || ''
        };
        
        if (!data.name) {
            this.showAlert('Lütfen deneyim adını girin.', 'error');
            return;
        }
        
        this.setButtonLoading(saveBtn, true);
        
        try {
            const payload = new FormData();
            payload.append('name', data.name);
            payload.append('start_date', this.convertDateForBackend(data.start_date));
            payload.append('end_date', this.convertDateForBackend(data.end_date));
            payload.append('description', data.description);
            // optional cover meta
            payload.append('cover_alt_text', formData.get('cover_alt_text') || '');
            payload.append('cover_title', formData.get('cover_title') || '');

            // attach cover if selected
            const coverInput = document.getElementById('experience_cover');
            if (coverInput && coverInput.files && coverInput.files.length > 0) {
                const file = coverInput.files[0];
                // Client-side validation similar to edit page
                const allowed = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp'];
                if (!allowed.includes(file.type)) {
                    this.showAlert('Geçersiz dosya türü. PNG, JPG veya WEBP dosyası yükleyin.', 'error');
                    this.setButtonLoading(saveBtn, false);
                    return;
                }

                payload.append('experience_cover', file);
            }

            const response = await fetch(`/backend/doctor/${this.doktorData.id}/add-experience`, {
                method: 'POST',
                body: payload
            });
            
            const result = await response.json();
            
            if (result.status === 201) {
                this.showAlert('Deneyim başarıyla eklendi.', 'success');
                this.closeAddExperienceModal();
                setTimeout(() => {
                    window.location.reload();
                }, 1500);
            } else {
                this.showAlert(result.message || 'Deneyim eklenirken hata oluştu.', 'error');
            }
        } catch (error) {
            console.error('Error saving experience:', error);
            this.showAlert('Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
        } finally {
            this.setButtonLoading(saveBtn, false);
        }
    }

    /**
     * Delete expertise
     */
    async deleteExpertise(duid, name) {
        if (!confirm(`"${name}" uzmanlık alanını silmek istediğinizden emin misiniz?`)) {
            return;
        }
        
        try {
            const response = await fetch(`/backend/doctor/${this.doktorData.id}/remove-expertise`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ duid: duid })
            });
            
            const result = await response.json();
            
            if (result.status === 201) {
                this.showAlert('Uzmanlık alanı başarıyla silindi.', 'success');
                setTimeout(() => {
                    window.location.reload();
                }, 1500);
            } else {
                this.showAlert(result.message || 'Uzmanlık alanı silinirken hata oluştu.', 'error');
            }
        } catch (error) {
            console.error('Error deleting expertise:', error);
            this.showAlert('Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
        }
    }

    /**
     * Delete experience
     */
    async deleteExperience(dtid, name) {
        if (!confirm(`"${name}" deneyimini silmek istediğinizden emin misiniz?`)) {
            return;
        }
        
        try {
            const response = await fetch(`/backend/doctor/${this.doktorData.id}/delete-experience`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ dtid: dtid })
            });
            
            const result = await response.json();
            
            if (result.status === 201) {
                this.showAlert('Deneyim başarıyla silindi.', 'success');
                setTimeout(() => {
                    window.location.reload();
                }, 1500);
            } else {
                this.showAlert(result.message || 'Deneyim silinirken hata oluştu.', 'error');
            }
        } catch (error) {
            console.error('Error deleting experience:', error);
            this.showAlert('Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
        }
    }

    /**
     * Delete doctor
     */
    async deleteDoctor() {
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        if (!confirmBtn) return;
        
        this.setButtonLoading(confirmBtn, true);
        
        try {
            const response = await fetch(`/backend/doctor/${this.doktorData.id}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });
            
            const data = await response.json();
            
            if (data.status === 201) {
                this.showAlert('Doktor başarıyla silindi.', 'success');
                setTimeout(() => {
                    window.location.href = '/panel/doktorlar';
                }, 1500);
            } else {
                this.showAlert(data.message || 'Doktor silinirken hata oluştu.', 'error');
                this.setButtonLoading(confirmBtn, false);
            }
        } catch (error) {
            console.error('Error deleting doctor:', error);
            this.showAlert('Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            this.setButtonLoading(confirmBtn, false);
        }
    }

    /**
     * Reset expertise form
     */
    resetExpertiseForm() {
        const form = document.getElementById('addExpertiseForm');
        if (form) {
            form.reset();
        }
    }

    /**
     * Reset experience form
     */
    resetExperienceForm() {
        const form = document.getElementById('addExperienceForm');
        if (form) {
            form.reset();
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
     * Format dates for better display
     */
    formatDates() {
        const dateElements = document.querySelectorAll('[data-date]');
        
        dateElements.forEach(element => {
            const dateValue = element.getAttribute('data-date');
            if (dateValue) {
                try {
                    const date = new Date(dateValue);
                    const formatted = date.toLocaleDateString('tr-TR', {
                        year: 'numeric',
                        month: 'long',
                        day: 'numeric'
                    });
                    element.textContent = formatted;
                } catch (error) {
                    console.warn('Invalid date format:', dateValue);
                }
            }
        });
    }

    /**
     * Setup copy to clipboard functionality
     */
    setupCopyToClipboard() {
        const copyButtons = document.querySelectorAll('[data-copy]');
        
        copyButtons.forEach(button => {
            button.addEventListener('click', (e) => {
                e.preventDefault();
                const textToCopy = button.getAttribute('data-copy');
                
                if (navigator.clipboard) {
                    navigator.clipboard.writeText(textToCopy).then(() => {
                        this.showAlert('Panoya kopyalandı!', 'success');
                    });
                } else {
                    // Fallback for older browsers
                    const textArea = document.createElement('textarea');
                    textArea.value = textToCopy;
                    document.body.appendChild(textArea);
                    textArea.select();
                    document.execCommand('copy');
                    document.body.removeChild(textArea);
                    this.showAlert('Panoya kopyalandı!', 'success');
                }
            });
        });
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

    /**
     * Download current zoomed image
     */
    downloadImage() {
        if (!this.currentZoomedImage) return;
        
        const link = document.createElement('a');
        link.href = this.currentZoomedImage;
        link.download = 'doktor-foto.jpg';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    }

    /**
     * Open current zoomed image in new tab
     */
    openImageInNewTab() {
        if (!this.currentZoomedImage) return;
        
        window.open(this.currentZoomedImage, '_blank');
    }

    /**
     * Handle experience cover selection and preview
     */
    handleExperienceCoverSelect(input, file, content, preview, previewImage, area) {
        const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp'];
        if (!allowedTypes.includes(file.type)) {
            this.showAlert('Geçersiz dosya türü. PNG, JPG veya WEBP dosyası yükleyin.', 'error');
            return;
        }
        const maxSize = 5 * 1024 * 1024; // 5MB
        if (file.size > maxSize) {
            this.showAlert('Dosya boyutu 5MB\'dan büyük olamaz.', 'error');
            return;
        }

        const dt = new DataTransfer();
        dt.items.add(file);
        if (input) input.files = dt.files;

        const reader = new FileReader();
        reader.onload = (e) => {
            if (previewImage) previewImage.src = e.target.result;
            if (content) content.style.display = 'none';
            if (preview) preview.style.display = 'flex';
            if (area) area.classList.add('has-file');
        };
        reader.readAsDataURL(file);
    }

    /**
     * Remove experience cover
     */
    removeExperienceCover(input, content, preview, area) {
        if (input) input.value = '';
        if (content) content.style.display = 'block';
        if (preview) preview.style.display = 'none';
        if (area) area.classList.remove('has-file');
    }

    /**
     * Setup edit experience handlers
     */
    setupEditExperienceHandlers() {
        // Edit experience button handlers
        document.addEventListener('click', (e) => {
            if (e.target.closest('.edit-experience-btn')) {
                const btn = e.target.closest('.edit-experience-btn');
                this.openEditExperienceModal(btn);
            }
        });

        // Update experience text button handler
        const updateTextBtn = document.getElementById('updateExperienceTextBtn');
        if (updateTextBtn) {
            updateTextBtn.addEventListener('click', () => {
                this.updateExperienceText();
            });
        }

        // Update experience picture button handler
        const updatePictureBtn = document.getElementById('updateExperiencePictureBtn');
        if (updatePictureBtn) {
            updatePictureBtn.addEventListener('click', () => {
                this.updateExperiencePicture();
            });
        }

        // Setup file upload for edit modal
        this.setupEditExperienceFileUpload();
    }

    /**
     * Open edit experience modal with data
     */
    openEditExperienceModal(btn) {
        const dtid = btn.dataset.dtid;
        const name = btn.dataset.name;
        const startDate = btn.dataset.startDate;
        const endDate = btn.dataset.endDate;
        const description = btn.dataset.description;
        const coverPath = btn.dataset.coverPath;
        const coverAltText = btn.dataset.coverAltText;
        const coverTitle = btn.dataset.coverTitle;

        // Populate text form fields
        document.getElementById('edit_dtid').value = dtid;
        document.getElementById('edit_experience_name').value = name;
        document.getElementById('edit_start_date').value = this.formatDateForInput(startDate);
        document.getElementById('edit_end_date').value = this.formatDateForInput(endDate);
        document.getElementById('edit_experience_description').value = description || '';

        // Set old values for change detection in text form (format dates consistently)
        document.getElementById('edit_old_name').value = name;
        document.getElementById('edit_old_start_date').value = this.formatDateForInput(startDate);
        document.getElementById('edit_old_end_date').value = this.formatDateForInput(endDate);
        document.getElementById('edit_old_description').value = description || '';

        // Populate picture form fields
        document.getElementById('edit_cover_experience_alt_text').value = coverAltText || '';
        document.getElementById('edit_cover_experience_title').value = coverTitle || '';

        // Set old values for change detection in picture form
        document.getElementById('edit_old_experience_cover_alt_text').value = coverAltText || '';
        document.getElementById('edit_old_experience_cover_title').value = coverTitle || '';

        // Show current cover image if exists
        const currentCoverContainer = document.getElementById('currentExperienceCover');
        if (coverPath) {
            currentCoverContainer.innerHTML = `
                <img src="${window.location.origin}/${coverPath}" alt="${coverAltText || ''}" title="${coverTitle || ''}" style="max-width: 100%; max-height: 100px; border-radius: var(--border-radius); box-shadow: var(--shadow-sm); object-fit: contain;">
            `;
        } else {
            currentCoverContainer.innerHTML = `
                <div class="no-media">
                    <i class="fas fa-image"></i>
                    <span>Kapak görseli yok</span>
                </div>
            `;
        }

        // Show modal
        const modal = document.getElementById('editExperienceModal');
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }

    /**
     * Format date for input field
     */
    formatDateForInput(dateValue) {
        if (!dateValue || dateValue === '' || dateValue.includes('0001-01-01')) {
            return '';
        }
        
        // If it's already in YYYY-MM-DD format
        if (/^\d{4}-\d{2}-\d{2}$/.test(dateValue)) {
            return dateValue;
        }
        
        // Try to parse and format
        const date = new Date(dateValue);
        if (!isNaN(date.getTime())) {
            return date.toISOString().split('T')[0];
        }
        
        return '';
    }

    /**
     * Show current cover image in edit modal
     */
    showCurrentCoverImage(coverPath, altText, title) {
        const area = document.getElementById('editExperienceCoverArea');
        const content = area.querySelector('.file-upload-content');
        const preview = area.querySelector('.file-preview');
        const previewImage = area.querySelector('.preview-image');
        
        previewImage.src = window.location.origin + '/' + coverPath;
        previewImage.alt = altText || '';
        previewImage.title = title || '';
        
        content.style.display = 'none';
        preview.style.display = 'flex';
        area.classList.add('has-file');
    }

    /**
     * Hide current cover image in edit modal
     */
    hideCurrentCoverImage() {
        const area = document.getElementById('editExperienceCoverArea');
        const content = area.querySelector('.file-upload-content');
        const preview = area.querySelector('.file-preview');
        
        content.style.display = 'block';
        preview.style.display = 'none';
        area.classList.remove('has-file');
    }

    /**
     * Setup file upload for edit experience modal
     */
    setupEditExperienceFileUpload() {
        const area = document.getElementById('editExperienceCoverArea');
        if (!area) return;

        const input = area.querySelector('.file-input');
        const content = area.querySelector('.file-upload-content');
        const preview = area.querySelector('.file-preview');
        const previewImage = area.querySelector('.preview-image');
        const removeBtn = area.querySelector('.file-remove');
        const browseLink = area.querySelector('.file-browse');

        // Click to browse
        browseLink.addEventListener('click', (e) => {
            e.preventDefault();
            input.click();
        });

        area.addEventListener('click', (e) => {
            if (e.target === area || e.target === content) {
                input.click();
            }
        });

        // Drag & drop events
        area.addEventListener('dragover', (e) => {
            e.preventDefault();
            area.classList.add('dragover');
        });

        area.addEventListener('dragleave', (e) => {
            e.preventDefault();
            if (!area.contains(e.relatedTarget)) {
                area.classList.remove('dragover');
            }
        });

        area.addEventListener('drop', (e) => {
            e.preventDefault();
            area.classList.remove('dragover');
            
            const files = e.dataTransfer.files;
            if (files.length > 0) {
                this.handleEditExperienceCoverSelect(input, files[0], content, preview, previewImage, area);
            }
        });

        // File input change
        input.addEventListener('change', (e) => {
            if (e.target.files.length > 0) {
                this.handleEditExperienceCoverSelect(input, e.target.files[0], content, preview, previewImage, area);
            }
        });

        // Remove file
        removeBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            this.removeEditExperienceCover(input, content, preview, area);
        });
    }

    /**
     * Handle edit experience cover selection
     */
    handleEditExperienceCoverSelect(input, file, content, preview, previewImage, area) {
        // Validate file type
        const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp'];
        if (!allowedTypes.includes(file.type)) {
            this.showAlert('Geçersiz dosya türü. PNG, JPG veya WEBP dosyası yükleyin.', 'error');
            return;
        }

        // Validate file size (5MB max)
        const maxSize = 5 * 1024 * 1024;
        if (file.size > maxSize) {
            this.showAlert('Dosya boyutu 5MB\'dan büyük olamaz.', 'error');
            return;
        }

        // Create file list for input
        const dt = new DataTransfer();
        dt.items.add(file);
        input.files = dt.files;

        // Show preview
        const reader = new FileReader();
        reader.onload = (e) => {
            previewImage.src = e.target.result;
            content.style.display = 'none';
            preview.style.display = 'flex';
            area.classList.add('has-file');
        };
        reader.readAsDataURL(file);
    }

    /**
     * Remove edit experience cover
     */
    removeEditExperienceCover(input, content, preview, area) {
        input.value = '';
        content.style.display = 'block';
        preview.style.display = 'none';
        area.classList.remove('has-file');
    }

    /**
     * Update experience text information
     */
    async updateExperienceText() {
        const form = document.getElementById('editExperienceTextForm');
        const updateBtn = document.getElementById('updateExperienceTextBtn');
        const dtid = document.getElementById('edit_dtid').value;

        if (!dtid) {
            this.showAlert('Deneyim ID bulunamadı.', 'error');
            return;
        }

        this.setButtonLoading(updateBtn, true);

        try {
            const data = {};

            // Add all form fields
            const inputs = form.querySelectorAll('input, textarea');
            inputs.forEach(input => {
                if (input.type === 'hidden') {
                    if (input.value === 'true') {
                        data[input.name] = true;
                    } else if (input.value === 'false') {
                        data[input.name] = false;
                    } else {
                        data[input.name] = input.value;
                    }
                } else {
                    data[input.name] = input.value;
                }
            });

            // Convert dates for backend (both new and old values for proper comparison)
            data.start_date = this.convertDateForBackend(data.start_date);
            data.end_date = this.convertDateForBackend(data.end_date);
            data.old_start_date = this.convertDateForBackend(data.old_start_date);
            data.old_end_date = this.convertDateForBackend(data.old_end_date);

            const response = await fetch(`/backend/doctor/${this.doktorData.id}/edit-experience`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data)
            });

            const result = await response.json();

            if (result.status === 201) {
                this.showAlert('Deneyim bilgileri başarıyla güncellendi.', 'success');
                this.closeEditExperienceModal();
                // Reload page to show updated data
                setTimeout(() => {
                    window.location.reload();
                }, 2000);
            } else {
                throw new Error(result.message || 'Güncelleme işlemi başarısız oldu.');
            }

        } catch (error) {
            console.error('Update experience text error:', error);
            this.showAlert(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
        } finally {
            this.setButtonLoading(updateBtn, false);
        }
    }

    /**
     * Update experience picture
     */
    async updateExperiencePicture() {
        const form = document.getElementById('editExperiencePictureForm');
        const updateBtn = document.getElementById('updateExperiencePictureBtn');
        const dtid = document.getElementById('edit_dtid').value;

        if (!dtid) {
            this.showAlert('Deneyim ID bulunamadı.', 'error');
            return;
        }

        this.setButtonLoading(updateBtn, true);

        try {
            const formData = new FormData();

            // Add all form fields
            const inputs = form.querySelectorAll('input');
            inputs.forEach(input => {
                if (input.type === 'file') {
                    if (input.files.length > 0) {
                        formData.append(input.name, input.files[0]);
                    }
                } else {
                    formData.append(input.name, input.value);
                }
            });

            formData.append("dtid", dtid);

            const response = await fetch(`/backend/doctor/${this.doktorData.id}/update-experience-picture`, {
                method: 'POST',
                body: formData
            });

            const result = await response.json();

            if (result.status === 201) {
                this.showAlert('Deneyim görseli başarıyla güncellendi.', 'success');
                this.closeEditExperienceModal();
                // Reload page to show updated data
                setTimeout(() => {
                    window.location.reload();
                }, 2000);
            } else {
                throw new Error(result.message || 'Görsel güncelleme işlemi başarısız oldu.');
            }

        } catch (error) {
            console.error('Update experience picture error:', error);
            this.showAlert(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
        } finally {
            this.setButtonLoading(updateBtn, false);
        }
    }

    /**
     * Close edit experience modal
     */
    closeEditExperienceModal() {
        const modal = document.getElementById('editExperienceModal');
        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
            // Reset forms
            document.getElementById('editExperienceTextForm').reset();
            document.getElementById('editExperiencePictureForm').reset();
            // Reset current cover display
            const currentCoverContainer = document.getElementById('currentExperienceCover');
            currentCoverContainer.innerHTML = '';
        }, 300);
    }
}

// Global functions for HTML onclick attributes
function toggleDoctorOptionsDropdown() {
    if (window.doktorViewManager) {
        window.doktorViewManager.toggleDropdown();
    }
}

function closeAddExpertiseModal() {
    if (window.doktorViewManager) {
        window.doktorViewManager.closeAddExpertiseModal();
    }
}

function closeAddExperienceModal() {
    if (window.doktorViewManager) {
        window.doktorViewManager.closeAddExperienceModal();
    }
}

function closeEditExperienceModal() {
    if (window.doktorViewManager) {
        window.doktorViewManager.closeEditExperienceModal();
    }
}

function closeDeleteModal() {
    if (window.doktorViewManager) {
        window.doktorViewManager.closeDeleteModal();
    }
}

function closeImageModal() {
    if (window.doktorViewManager) {
        window.doktorViewManager.closeImageModal();
    }
}

function closeAlert(alertId) {
    if (window.doktorViewManager) {
        window.doktorViewManager.closeAlert(alertId);
    }
}

function downloadImage() {
    if (window.doktorViewManager) {
        window.doktorViewManager.downloadImage();
    }
}

function openImageInNewTab() {
    if (window.doktorViewManager) {
        window.doktorViewManager.openImageInNewTab();
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.doktorViewManager = new DoktorViewManager();
});
