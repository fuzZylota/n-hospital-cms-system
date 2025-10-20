/**
 * Tedkik Ekle (Add Examination) Page JavaScript
 * Handles form submission, validation, and file uploads
 */

class TedkikFormHandler {
    constructor() {
        this.form = document.getElementById('tedkikForm');

        this.init();
    }

    init() {
        this.setupFormSubmission();
        this.setupFileUploads();
        this.setupFormValidation();
        this.setupToggleSwitches();
        this.setupNameToUrlSync();
    }

    /**
     * Setup form submission with direct submission
     */
    setupFormSubmission() {
        if (!this.form) return;

        this.form.addEventListener('submit', (e) => {
            // Validate form before submission
            if (!this.validateForm()) {
                e.preventDefault();
                return;
            }

            // Allow form to submit directly to the action URL
            // No preventDefault() - let the browser handle the submission
        });
    }

    /**
     * Setup file upload functionality
     */
    setupFileUploads() {
        const fileUploadAreas = document.querySelectorAll('.file-upload-area');

        fileUploadAreas.forEach(area => {
            const input = area.querySelector('.file-input');
            const content = area.querySelector('.file-upload-content');
            const preview = area.querySelector('.file-preview');
            const previewImage = preview?.querySelector('.preview-image');
            const removeBtn = preview?.querySelector('.file-remove');

            // Drag and drop events
            area.addEventListener('dragover', (e) => {
                e.preventDefault();
                area.classList.add('dragover');
            });

            area.addEventListener('dragleave', () => {
                area.classList.remove('dragover');
            });

            area.addEventListener('drop', (e) => {
                e.preventDefault();
                area.classList.remove('dragover');
                
                const files = e.dataTransfer.files;
                if (files.length > 0) {
                    this.handleFileSelect(input, files[0], content, preview, previewImage);
                }
            });

            // Click to upload
            area.addEventListener('click', () => {
                input.click();
            });

            // File input change
            input.addEventListener('change', (e) => {
                if (e.target.files.length > 0) {
                    this.handleFileSelect(input, e.target.files[0], content, preview, previewImage);
                }
            });

            // Remove file
            if (removeBtn) {
                removeBtn.addEventListener('click', (e) => {
                    e.stopPropagation();
                    this.removeFile(input, content, preview);
                });
            }
        });
    }

    /**
     * Handle file selection
     */
    handleFileSelect(input, file, content, preview, previewImage) {
        // Validate file type
        const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp'];
        if (!allowedTypes.includes(file.type)) {
            alert('Geçersiz dosya türü. Lütfen JPG, PNG veya WEBP formatında bir dosya seçin.');
            return;
        }

        // Validate file size (5MB max)
        const maxSize = 5 * 1024 * 1024; // 5MB
        if (file.size > maxSize) {
            alert('Dosya boyutu çok büyük. Maksimum 5MB dosya yükleyebilirsiniz.');
            return;
        }

        // Create preview
        const reader = new FileReader();
        reader.onload = (e) => {
            if (previewImage) {
                previewImage.src = e.target.result;
                previewImage.alt = file.name;
            }
            content.style.display = 'none';
            preview.style.display = 'block';
        };
        reader.readAsDataURL(file);

        // Update input
        input.files = new DataTransfer().files;
        const dataTransfer = new DataTransfer();
        dataTransfer.items.add(file);
        input.files = dataTransfer.files;
    }

    /**
     * Remove selected file
     */
    removeFile(input, content, preview) {
        input.value = '';
        content.style.display = 'block';
        preview.style.display = 'none';
    }

    /**
     * Setup form validation
     */
    setupFormValidation() {
        const inputs = this.form.querySelectorAll('input, textarea, select');
        
        inputs.forEach(input => {
            // Real-time validation
            input.addEventListener('blur', () => {
                this.validateField(input);
            });

            input.addEventListener('input', () => {
                // Clear error state on input
                this.setFieldState(input, true, '');
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
                const label = toggle.nextElementSibling;
                const text = label.querySelector('.toggle-text');
                if (text) {
                    text.textContent = toggle.checked ? 'Aktif' : 'Pasif';
                }
            };

            toggle.addEventListener('change', updateToggleText);
            updateToggleText(); // Initial state
        });
    }

    /**
     * Setup name to URL sync
     */
    setupNameToUrlSync() {
        const nameInput = document.getElementById('name');
        const urlInput = document.getElementById('url_name');

        if (!nameInput || !urlInput) return;

        nameInput.addEventListener('input', debounce(() => {
            if (urlInput.value === '' || urlInput.dataset.autoGenerated === 'true') {
                const urlName = this.generateUrlName(nameInput.value);
                urlInput.value = urlName;
                urlInput.dataset.autoGenerated = 'true';
            }
        }, 300));

        urlInput.addEventListener('input', () => {
            urlInput.dataset.autoGenerated = 'false';
        });
    }

    /**
     * Generate URL-friendly name
     */
    generateUrlName(name) {
        return name
            .toLowerCase()
            .replace(/[çÇ]/g, 'c')
            .replace(/[ğĞ]/g, 'g')
            .replace(/[ıİ]/g, 'i')
            .replace(/[öÖ]/g, 'o')
            .replace(/[şŞ]/g, 's')
            .replace(/[üÜ]/g, 'u')
            .replace(/[^a-z0-9\s-]/g, '')
            .replace(/\s+/g, '-')
            .replace(/-+/g, '-')
            .trim('-');
    }

    /**
     * Validate individual field
     */
    validateField(field) {
        const value = field.value.trim();
        const fieldName = field.name;
        let isValid = true;
        let errorMessage = '';

        // Required field validation
        if (field.hasAttribute('required') && !value) {
            isValid = false;
            errorMessage = 'Bu alan zorunludur.';
        }

        // Email validation
        if (fieldName === 'email' && value && !this.validateEmail(field)) {
            isValid = false;
            errorMessage = 'Geçerli bir e-posta adresi girin.';
        }

        // URL validation
        if (fieldName === 'website' && value && !this.validateUrl(field)) {
            isValid = false;
            errorMessage = 'Geçerli bir URL girin.';
        }

        // Phone validation
        if (fieldName === 'phone' && value && !this.validatePhone(field)) {
            isValid = false;
            errorMessage = 'Geçerli bir telefon numarası girin.';
        }

        this.setFieldState(field, isValid, errorMessage);
        return isValid;
    }

    /**
     * Validate email format
     */
    validateEmail(field) {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return emailRegex.test(field.value);
    }

    /**
     * Validate URL format
     */
    validateUrl(field) {
        try {
            new URL(field.value);
            return true;
        } catch {
            return false;
        }
    }

    /**
     * Validate phone format
     */
    validatePhone(field) {
        const phoneRegex = /^[\+]?[0-9\s\-\(\)]{10,}$/;
        return phoneRegex.test(field.value);
    }

    /**
     * Set field validation state
     */
    setFieldState(field, isValid, errorMessage) {
        const fieldGroup = field.closest('.form-group');
        const existingError = fieldGroup.querySelector('.field-error');
        
        // Remove existing error
        if (existingError) {
            existingError.remove();
        }

        // Update field appearance
        field.classList.remove('error', 'success');
        if (!isValid) {
            field.classList.add('error');
        } else if (field.value.trim()) {
            field.classList.add('success');
        }

        // Add error message
        if (!isValid && errorMessage) {
            const errorDiv = document.createElement('div');
            errorDiv.className = 'field-error';
            errorDiv.innerHTML = `<i class="icon-alert-circle"></i> ${errorMessage}`;
            fieldGroup.appendChild(errorDiv);
        }
    }

    /**
     * Validate entire form
     */
    validateForm() {
        const inputs = this.form.querySelectorAll('input[required], textarea[required], select[required]');
        let isFormValid = true;

        inputs.forEach(input => {
            if (!this.validateField(input)) {
                isFormValid = false;
            }
        });

        return isFormValid;
    }



    /**
     * Reset form
     */
    resetForm() {
        this.form.reset();
        
        // Reset file uploads
        const fileUploadAreas = document.querySelectorAll('.file-upload-area');
        fileUploadAreas.forEach(area => {
            const content = area.querySelector('.file-upload-content');
            const preview = area.querySelector('.file-preview');
            const input = area.querySelector('.file-input');
            
            if (content && preview && input) {
                content.style.display = 'block';
                preview.style.display = 'none';
                input.value = '';
            }
        });

        // Clear validation states
        const inputs = this.form.querySelectorAll('input, textarea, select');
        inputs.forEach(input => {
            this.setFieldState(input, true, '');
        });
    }
}

/**
 * Close modal function
 */
function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
    }
}

/**
 * Format bytes to human readable format
 */
function formatBytes(bytes, decimals = 2) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

/**
 * Debounce function
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

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new TedkikFormHandler();
});

// Export for potential external use
window.TedkikFormHandler = TedkikFormHandler;
