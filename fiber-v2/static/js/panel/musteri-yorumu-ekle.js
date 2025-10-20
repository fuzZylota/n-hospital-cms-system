/**
 * Musteri Yorumu Ekle (Testimonial Add) Page JavaScript
 * Handles form submission, file uploads, drag & drop, rating system, and validation
 */

class TestimonialFormHandler {
    constructor() {
        this.form = document.getElementById('testimonialForm');
        this.submitBtn = document.getElementById('submitBtn');
        this.init();
    }

    init() {
        this.setupFormSubmission();
        this.setupFileUploads();
        this.setupRatingSystem();
        this.setupCharacterCounter();
        this.setupFormValidation();
        this.setupToggleSwitches();
    }

    /**
     * Setup form submission (non-AJAX). Validate and allow native submit.
     */
    setupFormSubmission() {
        this.form.addEventListener('submit', (e) => {
            // Validate before submit; block submit if invalid
            if (!this.validateForm()) {
                e.preventDefault();
                return;
            }

            this.setLoadingState(true);
            // Allow natural form submission to the action URL
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
                    this.handleFileSelect(input, files[0], content, preview, previewImage);
                }
            });

            // File input change
            input.addEventListener('change', (e) => {
                if (e.target.files.length > 0) {
                    this.handleFileSelect(input, e.target.files[0], content, preview, previewImage);
                }
            });

            // Remove file
            removeBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                this.removeFile(input, content, preview);
            });
        });
    }

    /**
     * Handle file selection and preview
     */
    handleFileSelect(input, file, content, preview, previewImage) {
        // Validate file type
        const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp'];
        if (!allowedTypes.includes(file.type)) {
            this.showErrorModal('Geçersiz dosya türü. Lütfen JPG, PNG veya WebP formatında bir resim seçin.');
            return;
        }

        // Validate file size (2MB max)
        const maxSize = 2 * 1024 * 1024; // 2MB
        if (file.size > maxSize) {
            this.showErrorModal('Dosya boyutu 2MB\'dan büyük olamaz.');
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
        };
        reader.readAsDataURL(file);
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
     * Setup rating system
     */
    setupRatingSystem() {
        const ratingInput = document.getElementById('rating');
        const ratingStars = document.getElementById('ratingStars');
        const stars = ratingStars.querySelectorAll('.star');

        // Update stars when input changes
        const updateStars = (rating) => {
            const ratingValue = parseFloat(rating);
            stars.forEach((star, index) => {
                const starValue = index + 1;
                if (starValue <= ratingValue) {
                    star.classList.add('active');
                } else {
                    star.classList.remove('active');
                }
            });
        };

        // Handle star clicks
        stars.forEach((star, index) => {
            star.addEventListener('click', () => {
                const rating = index + 1;
                ratingInput.value = rating.toFixed(1);
                updateStars(rating);
            });

            star.addEventListener('mouseover', () => {
                const rating = index + 1;
                updateStars(rating);
            });
        });

        // Handle input field changes
        ratingInput.addEventListener('input', (e) => {
            let value = parseFloat(e.target.value);
            if (isNaN(value)) value = 0;
            if (value < 0) value = 0;
            if (value > 5) value = 5;
            
            e.target.value = value.toFixed(1);
            updateStars(value);
        });

        // Restore stars on mouse leave
        ratingStars.addEventListener('mouseleave', () => {
            updateStars(ratingInput.value);
        });

        // Initial setup
        updateStars(ratingInput.value);
    }

    /**
     * Setup character counter for content textarea
     */
    setupCharacterCounter() {
        const contentTextarea = document.getElementById('content');
        const counter = document.getElementById('contentCounter');
        const maxLength = 1000;

        const updateCounter = () => {
            const length = contentTextarea.value.length;
            counter.textContent = length;
            
            // Update counter styling based on length
            counter.parentElement.classList.remove('warning', 'error');
            if (length > maxLength * 0.8) {
                counter.parentElement.classList.add('warning');
            }
            if (length > maxLength) {
                counter.parentElement.classList.add('error');
            }
        };

        contentTextarea.addEventListener('input', updateCounter);
        updateCounter(); // Initial count
    }

    /**
     * Setup form validation
     */
    setupFormValidation() {
        const requiredFields = this.form.querySelectorAll('[required]');
        
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

        // Rating validation
        const ratingField = document.getElementById('rating');
        ratingField.addEventListener('blur', () => {
            this.validateRating(ratingField);
        });

        ratingField.addEventListener('input', () => {
            if (ratingField.classList.contains('error')) {
                this.validateRating(ratingField);
            }
        });

        // Content length validation
        const contentField = document.getElementById('content');
        contentField.addEventListener('input', () => {
            this.validateContentLength(contentField);
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
     * Validate rating field
     */
    validateRating(field) {
        const value = parseFloat(field.value);
        let isValid = true;
        let errorMessage = '';

        if (isNaN(value) || value < 0 || value > 5) {
            isValid = false;
            errorMessage = 'Puan 0-5 arasında olmalıdır.';
        }

        this.setFieldState(field, isValid, errorMessage);
        return isValid;
    }

    /**
     * Validate content length
     */
    validateContentLength(field) {
        const value = field.value;
        const maxLength = 1000;
        let isValid = true;
        let errorMessage = '';

        if (value.length > maxLength) {
            isValid = false;
            errorMessage = `Metin ${maxLength} karakterden uzun olamaz.`;
        }

        this.setFieldState(field, isValid, errorMessage);
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
     * Validate entire form
     */
    validateForm() {
        let isValid = true;
        
        // Validate required fields
        const requiredFields = this.form.querySelectorAll('[required]');
        requiredFields.forEach(field => {
            if (!this.validateField(field)) {
                isValid = false;
            }
        });

        // Validate rating
        const ratingField = document.getElementById('rating');
        if (!this.validateRating(ratingField)) {
            isValid = false;
        }

        // Validate content length
        const contentField = document.getElementById('content');
        if (!this.validateContentLength(contentField)) {
            isValid = false;
        }

        if (!isValid) {
            this.showErrorModal('Lütfen tüm alanları doğru şekilde doldurun.');
            
            // Scroll to first error
            const firstError = this.form.querySelector('.error');
            if (firstError) {
                firstError.scrollIntoView({ behavior: 'smooth', block: 'center' });
                firstError.focus();
            }
        }

        return isValid;
    }

    /**
     * Set loading state for form submission
     */
    setLoadingState(loading) {
        if (loading) {
            this.submitBtn.disabled = true;
            this.submitBtn.querySelector('.btn-text').style.opacity = '0';
            this.submitBtn.querySelector('.btn-loader').style.display = 'block';
            this.form.classList.add('form-loading');
        } else {
            this.submitBtn.disabled = false;
            this.submitBtn.querySelector('.btn-text').style.opacity = '1';
            this.submitBtn.querySelector('.btn-loader').style.display = 'none';
            this.form.classList.remove('form-loading');
        }
    }

    /**
     * Show success modal
     */
    showSuccessModal() {
        const modal = document.getElementById('successModal');
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
     * Reset form to initial state
     */
    resetForm() {
        this.form.reset();
        
        // Reset file uploads
        const filePreviews = this.form.querySelectorAll('.file-preview');
        const fileContents = this.form.querySelectorAll('.file-upload-content');
        
        filePreviews.forEach(preview => preview.style.display = 'none');
        fileContents.forEach(content => content.style.display = 'block');
        
        // Reset validation states
        const fields = this.form.querySelectorAll('.form-input, .form-textarea, .form-select');
        fields.forEach(field => {
            field.classList.remove('error', 'success');
        });
        
        // Remove error messages
        const errorElements = this.form.querySelectorAll('.field-error');
        errorElements.forEach(element => element.remove());
        
        // Reset rating stars
        const ratingInput = document.getElementById('rating');
        ratingInput.value = '5.0';
        this.setupRatingSystem();
        
        // Reset character counter
        this.setupCharacterCounter();
        
        // Reset toggle switches
        this.setupToggleSwitches();
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
        
        // If success modal, redirect to testimonials list
        if (modalId === 'successModal') {
            window.location.href = '/panel/musteri-yorumlari';
        }
    }, 300);
}

/**
 * Utility functions
 */
function formatBytes(bytes, decimals = 2) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
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
    new TestimonialFormHandler();
    
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

// Export for potential external use
window.TestimonialFormHandler = TestimonialFormHandler;
