/**
 * Brans Duzenle (Branch Edit) Page JavaScript
 * Handles general settings form, picture uploads, and picture deletion
 */

class BransEditHandler {
    constructor() {
        this.brid = window.pageData.brid;
        this.generalForm = document.getElementById('generalForm');
        this.pictureForm = document.getElementById('pictureForm');
        this.generalSubmitBtn = document.getElementById('generalSubmitBtn');
        this.pictureSubmitBtn = document.getElementById('pictureSubmitBtn');
        this.currentDeleteTarget = null;
        this.originalValues = {};
        
        this.init();
    }

    init() {
        this.initializeOriginalValues();
        this.setupGeneralForm();
        this.setupPictureForm();
        this.setupFileUploads();
        this.setupFormValidation();
        this.setupToggleSwitches();
        this.setupPictureDeletion();
        this.setupUrlGeneration();
        this.setupServicesSync();
        this.debugInitialValues();
    }

    /**
     * Store original values for change detection
     */
    initializeOriginalValues() {
        const altTextInput = document.getElementById('brans_media_alt_text');
        const titleInput = document.getElementById('brans_media_title');
        
        if (altTextInput) {
            this.originalValues.altText = altTextInput.value;
        }
        if (titleInput) {
            this.originalValues.title = titleInput.value;
        }
    }

    /**
     * Debug initial values to see what's being loaded
     */
    debugInitialValues() {
        const servicesInput = document.getElementById('services_input');
        const servicesField = document.getElementById('services');
        const oldServicesField = document.querySelector('input[name="old_services"]');
        
    }

    /**
     * Setup general settings form (AJAX JSON submission)
     */
    setupGeneralForm() {
        this.generalForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            if (!this.validateGeneralForm()) {
                return;
            }

            this.setGeneralLoadingState(true);

            try {
                // Process services input before collecting data
                this.processServicesInput();

                // Collect all form data
                const data = {};
                const allInputs = this.generalForm.querySelectorAll('input, textarea, select');
                
                allInputs.forEach(input => {
                    if (input.type === 'checkbox') {
                        data[input.name] = input.checked;
                    } else if (input.type === 'number') {
                        data[input.name] = parseFloat(input.value) || 0;
                    } else if (input.type === 'hidden') {
                        if (input.value === 'true') {
                            data[input.name] = true;
                        } else if (input.value === 'false') {
                            data[input.name] = false;
                        } else if (input.name.includes('old_') && document.querySelector(`[name="${input.name.replace('old_', '')}"]`)?.type === 'number') {
                            data[input.name] = parseFloat(input.value) || 0;
                        } else if (input.name === 'services' || input.name === 'old_services') {
                            // Parse JSON strings to arrays for services fields
                            try {
                                data[input.name] = JSON.parse(input.value);
                            } catch (e) {
                                console.error(`Failed to parse ${input.name}:`, e);
                                data[input.name] = [];
                            }
                        } else {
                            data[input.name] = input.value;
                        }
                    } else if (input.name === 'services_input') {
                        // Skip the textarea input, we'll use the processed hidden field
                        return;
                    } else {
                        data[input.name] = input.value;
                    }
                });


                const response = await fetch(`/backend/branch/${this.brid}/edit`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (result.status === 201) {
                    this.showSuccessModal('Branş bilgileri başarıyla güncellendi.');
                } else {
                    throw new Error(result.message || 'Güncelleme işlemi başarısız oldu.');
                }

            } catch (error) {
                console.error('General form submission error:', error);
                this.showErrorModal(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
            } finally {
                this.setGeneralLoadingState(false);
            }
        });
    }

    /**
     * Setup picture form (AJAX FormData submission)
     */
    setupPictureForm() {
        this.pictureForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            if (!this.validatePictureForm()) {
                return;
            }

            this.setPictureLoadingState(true);

            try {
                const formData = new FormData();
                
                // Add all form fields to FormData
                const inputs = this.pictureForm.querySelectorAll('input');
                inputs.forEach(input => {
                    if (input.type === 'file') {
                        if (input.files.length > 0) {
                            formData.append(input.name, input.files[0]);
                        }
                    } else {
                        formData.append(input.name, input.value);
                    }
                });

                const response = await fetch(`/backend/branch/${this.brid}/update-picture`, {
                    method: 'POST',
                    body: formData
                });

                const result = await response.json();

                if (result.status === 201) {
                    this.showSuccessModal('Branş görseli başarıyla güncellendi.');
                    // Reload page to show updated picture
                    setTimeout(() => {
                        window.location.reload();
                    }, 2000);
                } else {
                    throw new Error(result.message || 'Görsel güncelleme işlemi başarısız oldu.');
                }

            } catch (error) {
                console.error('Picture form submission error:', error);
                this.showErrorModal(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
            } finally {
                this.setPictureLoadingState(false);
            }
        });
    }

    /**
     * Setup file upload areas with drag & drop functionality
     */
    setupFileUploads() {
        const fileUploadAreas = document.querySelectorAll('.file-upload-area');
        
        fileUploadAreas.forEach(area => {
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
                    this.handleFileSelect(input, files[0], content, preview, previewImage, area);
                }
            });

            // File input change
            input.addEventListener('change', (e) => {
                if (e.target.files.length > 0) {
                    this.handleFileSelect(input, e.target.files[0], content, preview, previewImage, area);
                }
            });

            // Remove file
            removeBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                this.removeFile(input, content, preview, area);
            });
        });
    }

    /**
     * Handle file selection and preview
     */
    handleFileSelect(input, file, content, preview, previewImage, area) {
        // Validate file type
        const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp'];
        if (!allowedTypes.includes(file.type)) {
            this.showErrorModal('Geçersiz dosya türü. PNG, JPG veya WEBP dosyası yükleyin.');
            return;
        }

        // Validate file size (5MB max)
        const maxSize = 5 * 1024 * 1024; // 5MB
        if (file.size > maxSize) {
            this.showErrorModal('Dosya boyutu 5MB\'dan büyük olamaz.');
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
     * Remove selected file
     */
    removeFile(input, content, preview, area) {
        input.value = '';
        content.style.display = 'block';
        preview.style.display = 'none';
        area.classList.remove('has-file');
    }

    /**
     * Setup form validation
     */
    setupFormValidation() {
        // General form validation
        const requiredFields = this.generalForm.querySelectorAll('[required]');
        requiredFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validateField(field);
            });
            
            field.addEventListener('input', () => {
                if (field.classList.contains('error')) {
                    this.validateField(field);
                }
            });
        });

        // Email validation
        const emailFields = this.generalForm.querySelectorAll('input[type="email"]');
        emailFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validateEmail(field);
            });
        });

        // Phone validation
        const phoneFields = this.generalForm.querySelectorAll('input[type="tel"]');
        phoneFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validatePhone(field);
            });
        });
    }

    /**
     * Setup toggle switches
     */
    setupToggleSwitches() {
        const toggles = document.querySelectorAll('.toggle-switch input[type="checkbox"]');
        
        toggles.forEach(toggle => {
            const updateToggleText = () => {
                const textElement = toggle.parentNode.querySelector('.toggle-text');
                if (textElement) {
                    textElement.textContent = toggle.checked ? 'Aktif' : 'Pasif';
                }
            };
            
            toggle.addEventListener('change', updateToggleText);
            updateToggleText(); // Initial state
        });
    }

    /**
     * Setup picture deletion functionality
     */
    setupPictureDeletion() {
        const deleteButtons = document.querySelectorAll('.btn-delete-media');
        const confirmBtn = document.getElementById('confirmDeletePictureBtn');
        
        deleteButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                const brid = btn.dataset.brid;
                
                this.currentDeleteTarget = { brid: brid, button: btn };
                this.showDeletePictureModal();
            });
        });

        confirmBtn.addEventListener('click', async () => {
            if (!this.currentDeleteTarget) return;
            
            await this.deletePicture(this.currentDeleteTarget);
        });
    }

    /**
     * Setup URL generation from name
     */
    setupUrlGeneration() {
        const nameInput = document.getElementById('name');
        const urlInput = document.getElementById('url_name');
        
        if (nameInput && urlInput) {
            nameInput.addEventListener('input', () => {
                if (!urlInput.value || urlInput.dataset.autoGenerated === 'true') {
                    const urlFriendly = this.generateUrlFriendlyString(nameInput.value);
                    urlInput.value = urlFriendly;
                    urlInput.dataset.autoGenerated = 'true';
                }
            });
            
            urlInput.addEventListener('input', () => {
                // Mark as manually edited if user types
                if (urlInput.value !== this.generateUrlFriendlyString(nameInput.value)) {
                    urlInput.dataset.autoGenerated = 'false';
                }
            });
        }
    }

    /**
     * Setup services input synchronization
     */
    setupServicesSync() {
        const servicesInput = document.getElementById('services_input');
        if (!servicesInput) return;

        // Sync on input change with debouncing
        const debouncedSync = debounce(() => {
            this.processServicesInput();
        }, 500);

        servicesInput.addEventListener('input', debouncedSync);
        servicesInput.addEventListener('paste', () => {
            // Process immediately after paste
            setTimeout(() => {
                this.processServicesInput();
            }, 100);
        });
    }

    /**
     * Process services input and convert to JSON array
     */
    processServicesInput() {
        const servicesInput = document.getElementById('services_input');
        const servicesField = document.getElementById('services');
        
        if (!servicesInput || !servicesField) {
            return;
        }

        const servicesText = servicesInput.value.trim();
        
        if (!servicesText) {
            // If empty, set empty JSON array
            servicesField.value = '[]';
            return;
        }

        // Split by newlines and process each service
        const servicesArray = servicesText
            .split('\n')
            .map(service => service.trim())
            .filter(service => service.length > 0) // Remove empty lines
            .map(service => {
                // Clean up the service name
                return service
                    .replace(/^[-•\*\+]\s*/, '') // Remove bullet points
                    .replace(/\s+/g, ' ') // Normalize whitespace
                    .trim();
            })
            .filter(service => service.length > 0); // Remove empty services after cleaning

        // Convert to JSON string and set as hidden input value
        const servicesJson = JSON.stringify(servicesArray);
        servicesField.value = servicesJson;
    }

    /**
     * Generate URL-friendly string from Turkish text
     */
    generateUrlFriendlyString(text) {
        const turkishMap = {
            'ç': 'c', 'Ç': 'C',
            'ğ': 'g', 'Ğ': 'G',
            'ı': 'i', 'I': 'I',
            'İ': 'I', 'i': 'i',
            'ö': 'o', 'Ö': 'O',
            'ş': 's', 'Ş': 'S',
            'ü': 'u', 'Ü': 'U'
        };
        
        return text
            .split('')
            .map(char => turkishMap[char] || char)
            .join('')
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, '-')
            .replace(/^-+|-+$/g, '')
            .substring(0, 50);
    }

    /**
     * Delete picture file
     */
    async deletePicture(target) {
        this.setDeletePictureLoadingState(true);

        try {
            const response = await fetch(`/backend/branch/${target.brid}/delete-picture`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            const result = await response.json();

            if (result.status === 201) {
                this.showSuccessModal('Branş görseli başarıyla silindi.');
                // Remove the media item from UI
                const mediaItem = target.button.closest('.media-item');
                if (mediaItem) {
                    mediaItem.classList.add('deleting');
                    setTimeout(() => {
                        window.location.reload();
                    }, 1000);
                }
                this.closeModal('deletePictureModal');
            } else {
                throw new Error(result.message || 'Görsel silme işlemi başarısız oldu.');
            }

        } catch (error) {
            console.error('Picture deletion error:', error);
            this.showErrorModal(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
        } finally {
            this.setDeletePictureLoadingState(false);
        }
    }

    /**
     * Validate individual field
     */
    validateField(field) {
        const value = field.value.trim();
        let isValid = true;
        let errorMessage = '';

        // Required field validation
        if (field.hasAttribute('required') && !value) {
            isValid = false;
            errorMessage = 'Bu alan zorunludur.';
        }

        // Minimum length validation
        if (field.hasAttribute('minlength') && value.length < field.getAttribute('minlength')) {
            isValid = false;
            errorMessage = `En az ${field.getAttribute('minlength')} karakter olmalıdır.`;
        }

        this.setFieldState(field, isValid, errorMessage);
        return isValid;
    }

    /**
     * Validate email field
     */
    validateEmail(field) {
        const email = field.value.trim();
        if (!email) return true; // Not required
        
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        const isValid = emailRegex.test(email);
        
        this.setFieldState(field, isValid, isValid ? '' : 'Geçerli bir e-posta adresi girin.');
        return isValid;
    }

    /**
     * Validate phone field
     */
    validatePhone(field) {
        const phone = field.value.trim();
        if (!phone) return true; // Not required
        
        const phoneRegex = /^[\+]?[0-9\s\-\(\)]{10,}$/;
        const isValid = phoneRegex.test(phone);
        
        this.setFieldState(field, isValid, isValid ? '' : 'Geçerli bir telefon numarası girin.');
        return isValid;
    }

    /**
     * Set field validation state
     */
    setFieldState(field, isValid, errorMessage) {
        const fieldGroup = field.closest('.form-group');
        let errorElement = fieldGroup.querySelector('.field-error');

        // Remove existing error
        if (errorElement) {
            errorElement.remove();
        }

        // Update field classes
        field.classList.remove('error', 'success');
        field.classList.add(isValid ? 'success' : 'error');

        // Add error message
        if (!isValid && errorMessage) {
            errorElement = document.createElement('div');
            errorElement.className = 'field-error';
            errorElement.innerHTML = `<i class="fas fa-exclamation-circle"></i> ${errorMessage}`;
            fieldGroup.appendChild(errorElement);
        }
    }

    /**
     * Validate general form
     */
    validateGeneralForm() {
        let isValid = true;
        
        // Validate required fields
        const requiredFields = this.generalForm.querySelectorAll('[required]');
        requiredFields.forEach(field => {
            if (!this.validateField(field)) {
                isValid = false;
            }
        });

        // Validate email fields
        const emailFields = this.generalForm.querySelectorAll('input[type="email"]');
        emailFields.forEach(field => {
            if (!this.validateEmail(field)) {
                isValid = false;
            }
        });

        // Validate phone fields
        const phoneFields = this.generalForm.querySelectorAll('input[type="tel"]');
        phoneFields.forEach(field => {
            if (!this.validatePhone(field)) {
                isValid = false;
            }
        });

        if (!isValid) {
            this.showErrorModal('Lütfen tüm alanları doğru şekilde doldurun.');
            
            // Scroll to first error
            const firstError = this.generalForm.querySelector('.error');
            if (firstError) {
                firstError.scrollIntoView({ behavior: 'smooth', block: 'center' });
                firstError.focus();
            }
        }

        return isValid;
    }

    /**
     * Validate picture form
     */
    validatePictureForm() {
        // Check if file is selected or metadata has changed
        const fileInput = this.pictureForm.querySelector('input[type="file"]');
        const altTextInput = document.getElementById('brans_media_alt_text');
        const titleInput = document.getElementById('brans_media_title');
        
        let hasFile = fileInput && fileInput.files.length > 0;
        let hasMetadataChange = false;
        
        // Check if alt text or title has changed
        if (altTextInput && altTextInput.value !== this.originalValues.altText) {
            hasMetadataChange = true;
        }
        if (titleInput && titleInput.value !== this.originalValues.title) {
            hasMetadataChange = true;
        }
        
        if (!hasFile && !hasMetadataChange) {
            this.showErrorModal('Lütfen bir dosya seçin veya görsel bilgilerini değiştirin.');
            return false;
        }
        
        return true;
    }

    /**
     * Set loading state for general form
     */
    setGeneralLoadingState(loading) {
        if (loading) {
            this.generalSubmitBtn.disabled = true;
            this.generalSubmitBtn.querySelector('.btn-text').style.opacity = '0';
            this.generalSubmitBtn.querySelector('.btn-loader').style.display = 'block';
            this.generalForm.classList.add('form-loading');
        } else {
            this.generalSubmitBtn.disabled = false;
            this.generalSubmitBtn.querySelector('.btn-text').style.opacity = '1';
            this.generalSubmitBtn.querySelector('.btn-loader').style.display = 'none';
            this.generalForm.classList.remove('form-loading');
        }
    }

    /**
     * Set loading state for picture form
     */
    setPictureLoadingState(loading) {
        if (loading) {
            this.pictureSubmitBtn.disabled = true;
            this.pictureSubmitBtn.querySelector('.btn-text').style.opacity = '0';
            this.pictureSubmitBtn.querySelector('.btn-loader').style.display = 'block';
            this.pictureForm.classList.add('form-loading');
        } else {
            this.pictureSubmitBtn.disabled = false;
            this.pictureSubmitBtn.querySelector('.btn-text').style.opacity = '1';
            this.pictureSubmitBtn.querySelector('.btn-loader').style.display = 'none';
            this.pictureForm.classList.remove('form-loading');
        }
    }

    /**
     * Set loading state for delete picture button
     */
    setDeletePictureLoadingState(loading) {
        const confirmBtn = document.getElementById('confirmDeletePictureBtn');
        if (loading) {
            confirmBtn.disabled = true;
            confirmBtn.querySelector('.btn-text').style.opacity = '0';
            confirmBtn.querySelector('.btn-loader').style.display = 'block';
        } else {
            confirmBtn.disabled = false;
            confirmBtn.querySelector('.btn-text').style.opacity = '1';
            confirmBtn.querySelector('.btn-loader').style.display = 'none';
        }
    }

    /**
     * Show success modal
     */
    showSuccessModal(message) {
        const modal = document.getElementById('successModal');
        const messageElement = document.getElementById('successMessage');
        messageElement.textContent = message;
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }

    /**
     * Show error modal
     */
    showErrorModal(message) {
        const modal = document.getElementById('errorModal');
        const messageElement = document.getElementById('errorMessage');
        messageElement.textContent = message;
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }

    /**
     * Show delete picture modal
     */
    showDeletePictureModal() {
        const modal = document.getElementById('deletePictureModal');
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }

    /**
     * Close modal
     */
    closeModal(modalId) {
        const modal = document.getElementById(modalId);
        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
            if (modalId === 'deletePictureModal') {
                this.currentDeleteTarget = null;
            }
        }, 300);
    }
}

/**
 * Modal management functions
 */
function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    modal.classList.remove('show');
    setTimeout(() => {
        modal.style.display = 'none';
    }, 300);
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new BransEditHandler();
    
    // Close modals when clicking outside
    document.addEventListener('click', (e) => {
        if (e.target.classList.contains('modal')) {
            const modalId = e.target.id;
            closeModal(modalId);
        }
    });
    
    // Close modals with Escape key
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            const visibleModal = document.querySelector('.modal.show');
            if (visibleModal) {
                closeModal(visibleModal.id);
            }
        }
    });
});

/**
 * Utility functions
 */
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

// Export for potential external use
window.BransEditHandler = BransEditHandler;
