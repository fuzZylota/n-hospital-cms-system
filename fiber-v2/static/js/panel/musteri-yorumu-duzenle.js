/**
 * Musteri Yorumu Duzenle (Testimonial Edit) Page JavaScript
 * Handles form submission, file uploads, drag & drop, rating system, and validation
 */

class TestimonialEditHandler {
    constructor() {
        this.generalForm = document.getElementById('generalForm');
        this.pictureForm = document.getElementById('pictureForm');
        this.generalSubmitBtn = document.getElementById('generalSubmitBtn');
        this.pictureSubmitBtn = document.getElementById('pictureSubmitBtn');
        this.tid = window.pageData?.tid || null;
        this.currentRating = window.pageData?.currentRating || 0;
        this.init();
    }

    init() {
        this.setupGeneralForm();
        this.setupPictureForm();
        this.setupFileUploads();
        this.setupFormValidation();
        this.setupToggleSwitches();
        this.setupRatingSystem();
        this.setupCharacterCounter();
        this.setupPictureDeletion();
        this.initializeOriginalValues();
    }

    /**
     * Setup general information form submission
     */
    setupGeneralForm() {
        this.generalForm.addEventListener('submit', async (e) => {
            e.preventDefault();

            if (!this.validateGeneralForm()) {
                return;
            }

            this.setGeneralLoadingState(true);

            try {
                const formData = new FormData(this.generalForm);
                
                const response = await fetch(`/backend/testimonial/${this.tid}/edit`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        first_name: formData.get('first_name'),
                        old_first_name: formData.get('old_first_name'),
                        last_name: formData.get('last_name'),
                        old_last_name: formData.get('old_last_name'),
                        occupation: formData.get('occupation'),
                        old_occupation: formData.get('old_occupation'),
                        content: formData.get('content'),
                        old_content: formData.get('old_content'),
                        rating: parseFloat(formData.get('rating')),
                        old_rating: parseFloat(formData.get('old_rating')),
                        is_active: formData.get('is_active') === 'on',
                        old_is_active: formData.get('old_is_active') === 'true'
                    })
                });

                const result = await response.json();

                if (result.status === 201) {
                    this.showSuccessModal('Müşteri yorumu başarıyla güncellendi.');
                    // Update old values for next comparison
                    this.updateOldValues();
                } else {
                    this.showErrorModal(result.message || 'Güncelleme sırasında bir hata oluştu.');
                }
            } catch (error) {
                console.error('Error updating testimonial:', error);
                this.showErrorModal('Sunucu hatası: Lütfen daha sonra tekrar deneyin.');
            } finally {
                this.setGeneralLoadingState(false);
            }
        });
    }

    /**
     * Setup picture form submission
     */
    setupPictureForm() {
        this.pictureForm.addEventListener('submit', async (e) => {
            e.preventDefault();

            if (!this.validatePictureForm()) {
                return;
            }

            this.setPictureLoadingState(true);

            try {
                const formData = new FormData(this.pictureForm);
                
                const response = await fetch(`/backend/testimonial/${this.tid}/update-picture`, {
                    method: 'POST',
                    body: formData
                });

                const result = await response.json();

                if (result.status === 201) {
                    this.showSuccessModal('Müşteri fotoğrafı başarıyla güncellendi.');
                    setTimeout(() => {
                        window.location.reload();
                    }, 1500);
                } else {
                    this.showErrorModal(result.message || 'Fotoğraf güncellemesi sırasında bir hata oluştu.');
                }
            } catch (error) {
                console.error('Error updating picture:', error);
                this.showErrorModal('Sunucu hatası: Lütfen daha sonra tekrar deneyin.');
            } finally {
                this.setPictureLoadingState(false);
            }
        });
    }

    /**
     * Setup file upload functionality with drag & drop
     */
    setupFileUploads() {
        const fileUploadAreas = document.querySelectorAll('.file-upload-area');
        
        fileUploadAreas.forEach(area => {
            const targetId = area.dataset.target;
            const input = document.getElementById(targetId);
            const content = area.querySelector('.file-upload-content');
            const preview = area.querySelector('.file-preview');
            const previewImage = preview?.querySelector('.preview-image');
            
            if (!input || !content || !preview || !previewImage) return;

            // File input change
            input.addEventListener('change', (e) => {
                const file = e.target.files[0];
                if (file) {
                    this.handleFileSelect(input, file, content, preview, previewImage, area);
                }
            });

            // Drag & drop
            area.addEventListener('dragover', (e) => {
                e.preventDefault();
                area.classList.add('dragover');
            });

            area.addEventListener('dragleave', (e) => {
                e.preventDefault();
                area.classList.remove('dragover');
            });

            area.addEventListener('drop', (e) => {
                e.preventDefault();
                area.classList.remove('dragover');
                
                const file = e.dataTransfer.files[0];
                if (file && file.type.startsWith('image/')) {
                    input.files = e.dataTransfer.files;
                    this.handleFileSelect(input, file, content, preview, previewImage, area);
                }
            });

            // Remove file
            const removeBtn = preview.querySelector('.file-remove');
            if (removeBtn) {
                removeBtn.addEventListener('click', () => {
                    this.removeFile(input, content, preview, area);
                });
            }
        });
    }

    handleFileSelect(input, file, content, preview, previewImage, area) {
        // Validate file type
        if (!file.type.startsWith('image/')) {
            this.showErrorModal('Lütfen sadece resim dosyası seçin.');
            return;
        }

        // Validate file size (5MB max)
        const maxSize = 5 * 1024 * 1024; // 5MB
        if (file.size > maxSize) {
            this.showErrorModal('Dosya boyutu 5MB\'dan büyük olamaz.');
            return;
        }

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
        // Real-time validation for general form
        const generalFields = ['first_name', 'last_name', 'occupation', 'content', 'rating'];
        generalFields.forEach(fieldName => {
            const field = document.getElementById(fieldName);
            if (field) {
                field.addEventListener('input', () => this.validateField(field));
                field.addEventListener('blur', () => this.validateField(field));
            }
        });
    }

    /**
     * Setup toggle switches
     */
    setupToggleSwitches() {
        const toggles = document.querySelectorAll('.toggle-switch input[type="checkbox"]');
        
        toggles.forEach(toggle => {
            const updateToggleText = () => {
                const label = toggle.closest('.toggle-switch').querySelector('.toggle-text');
                if (label) {
                    label.textContent = toggle.checked ? 'Aktif' : 'Pasif';
                }
            };

            toggle.addEventListener('change', updateToggleText);
            updateToggleText(); // Initialize
        });
    }

    /**
     * Setup rating system with stars
     */
    setupRatingSystem() {
        const ratingInput = document.getElementById('rating');
        const starDisplay = document.getElementById('starDisplay');
        
        if (!ratingInput || !starDisplay) return;

        // Create stars
        for (let i = 1; i <= 5; i++) {
            const star = document.createElement('i');
            star.className = 'fas fa-star star';
            star.dataset.rating = i;
            starDisplay.appendChild(star);
        }

        const stars = starDisplay.querySelectorAll('.star');

        // Update stars display
        const updateStars = (rating) => {
            stars.forEach((star, index) => {
                const starRating = index + 1;
                star.classList.remove('filled', 'half');
                
                if (rating >= starRating) {
                    star.classList.add('filled');
                } else if (rating >= starRating - 0.5) {
                    star.classList.add('half');
                }
            });
        };

        // Initialize stars
        updateStars(this.currentRating);

        // Rating input change
        ratingInput.addEventListener('input', (e) => {
            const rating = parseFloat(e.target.value) || 0;
            updateStars(rating);
        });

        // Star click
        stars.forEach(star => {
            star.addEventListener('click', () => {
                const rating = parseInt(star.dataset.rating);
                ratingInput.value = rating;
                updateStars(rating);
                this.validateField(ratingInput);
            });
        });
    }

    /**
     * Setup character counter for content
     */
    setupCharacterCounter() {
        const contentField = document.getElementById('content');
        const counter = document.getElementById('contentCounter');
        
        if (!contentField || !counter) return;

        const updateCounter = () => {
            const length = contentField.value.length;
            const maxLength = 1000;
            
            counter.textContent = length;
            counter.parentElement.classList.remove('warning', 'error');
            
            if (length > maxLength * 0.9) {
                counter.parentElement.classList.add('warning');
            }
            if (length > maxLength) {
                counter.parentElement.classList.add('error');
            }
        };

        contentField.addEventListener('input', updateCounter);
        updateCounter(); // Initialize
    }

    /**
     * Initialize original values for change detection
     */
    initializeOriginalValues() {
        const altTextInput = document.getElementById('customer_picture_alt_text');
        const titleInput = document.getElementById('customer_picture_title');
        
        if (altTextInput) {
            altTextInput.setAttribute('data-original-value', altTextInput.value);
        }
        if (titleInput) {
            titleInput.setAttribute('data-original-value', titleInput.value);
        }
    }

    /**
     * Setup picture deletion
     */
    setupPictureDeletion() {
        const deleteButtons = document.querySelectorAll('.btn-delete-media');
        const confirmBtn = document.getElementById('confirmDeletePictureBtn');
        
        deleteButtons.forEach(button => {
            button.addEventListener('click', () => {
                this.currentDeleteTarget = button;
                this.showDeletePictureModal();
            });
        });

        if (confirmBtn) {
            confirmBtn.addEventListener('click', async () => {
                if (this.currentDeleteTarget) {
                    await this.deletePicture(this.currentDeleteTarget);
                }
            });
        }
    }

    async deletePicture(target) {
        this.setDeletePictureLoadingState(true);

        try {
            const response = await fetch(`/backend/testimonial/${this.tid}/delete-picture`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            const result = await response.json();

            if (result.status === 201) {
                this.closeModal('deletePictureModal');
                this.showSuccessModal('Fotoğraf başarıyla silindi.');
                setTimeout(() => {
                    window.location.reload();
                }, 1500);
            } else {
                this.showErrorModal(result.message || 'Fotoğraf silinirken bir hata oluştu.');
            }
        } catch (error) {
            console.error('Error deleting picture:', error);
            this.showErrorModal('Sunucu hatası: Lütfen daha sonra tekrar deneyin.');
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

        switch (field.id) {
            case 'first_name':
            case 'last_name':
            case 'occupation':
                if (!value) {
                    isValid = false;
                    errorMessage = 'Bu alan zorunludur.';
                } else if (value.length < 2) {
                    isValid = false;
                    errorMessage = 'En az 2 karakter olmalıdır.';
                }
                break;
            
            case 'content':
                if (!value) {
                    isValid = false;
                    errorMessage = 'Yorum metni zorunludur.';
                } else if (value.length < 10) {
                    isValid = false;
                    errorMessage = 'En az 10 karakter olmalıdır.';
                } else if (value.length > 1000) {
                    isValid = false;
                    errorMessage = 'En fazla 1000 karakter olabilir.';
                }
                break;
            
            case 'rating':
                const rating = parseFloat(value);
                if (isNaN(rating) || rating < 0 || rating > 5) {
                    isValid = false;
                    errorMessage = '0-5 arasında bir değer giriniz.';
                }
                break;
        }

        this.setFieldState(field, isValid, errorMessage);
        return isValid;
    }

    setFieldState(field, isValid, errorMessage) {
        field.classList.remove('error', 'success');
        
        // Remove existing error message
        const existingError = field.parentElement.querySelector('.field-error');
        if (existingError) {
            existingError.remove();
        }

        if (isValid) {
            field.classList.add('success');
        } else {
            field.classList.add('error');
            
            // Add error message
            if (errorMessage) {
                const errorDiv = document.createElement('div');
                errorDiv.className = 'field-error';
                errorDiv.style.color = 'var(--danger-color)';
                errorDiv.style.fontSize = '12px';
                errorDiv.style.marginTop = '4px';
                errorDiv.textContent = errorMessage;
                field.parentElement.appendChild(errorDiv);
            }
        }
    }

    /**
     * Validate general form
     */
    validateGeneralForm() {
        const requiredFields = ['first_name', 'last_name', 'occupation', 'content', 'rating'];
        let isValid = true;

        requiredFields.forEach(fieldId => {
            const field = document.getElementById(fieldId);
            if (field && !this.validateField(field)) {
                isValid = false;
            }
        });

        return isValid;
    }

    /**
     * Validate picture form
     */
    validatePictureForm() {
        const fileInput = document.getElementById('customer_picture_path');
        const altTextInput = document.getElementById('customer_picture_alt_text');
        const titleInput = document.getElementById('customer_picture_title');
        
        // Check if there are any changes to alt text or title
        const hasAltTextChange = altTextInput && altTextInput.value !== altTextInput.getAttribute('data-original-value');
        const hasTitleChange = titleInput && titleInput.value !== titleInput.getAttribute('data-original-value');
        const hasFileChange = fileInput.files.length > 0;
        
        // Allow submission if there are changes to alt text, title, or file
        if (!hasAltTextChange && !hasTitleChange && !hasFileChange) {
            this.showErrorModal('Lütfen en az bir değişiklik yapın (fotoğraf, alt metin veya başlık).');
            return false;
        }

        return true;
    }

    /**
     * Update old values after successful update
     */
    updateOldValues() {
        const formData = new FormData(this.generalForm);
        
        // Update hidden old value inputs
        const oldInputs = {
            'old_first_name': formData.get('first_name'),
            'old_last_name': formData.get('last_name'),
            'old_occupation': formData.get('occupation'),
            'old_content': formData.get('content'),
            'old_rating': formData.get('rating'),
            'old_is_active': formData.get('is_active') === 'on' ? 'true' : 'false'
        };

        Object.entries(oldInputs).forEach(([name, value]) => {
            const input = this.generalForm.querySelector(`input[name="${name}"]`);
            if (input) {
                input.value = value;
            }
        });
    }

    /**
     * Loading state management
     */
    setGeneralLoadingState(loading) {
        const btn = this.generalSubmitBtn;
        const btnText = btn.querySelector('.btn-text');
        const btnLoader = btn.querySelector('.btn-loader');

        btn.disabled = loading;
        btnText.style.opacity = loading ? '0' : '1';
        btnLoader.style.display = loading ? 'block' : 'none';
        
        if (loading) {
            this.generalForm.classList.add('form-loading');
        } else {
            this.generalForm.classList.remove('form-loading');
        }
    }

    setPictureLoadingState(loading) {
        const btn = this.pictureSubmitBtn;
        const btnText = btn.querySelector('.btn-text');
        const btnLoader = btn.querySelector('.btn-loader');

        btn.disabled = loading;
        btnText.style.opacity = loading ? '0' : '1';
        btnLoader.style.display = loading ? 'block' : 'none';
        
        if (loading) {
            this.pictureForm.classList.add('form-loading');
        } else {
            this.pictureForm.classList.remove('form-loading');
        }
    }

    setDeletePictureLoadingState(loading) {
        const btn = document.getElementById('confirmDeletePictureBtn');
        const btnText = btn.querySelector('.btn-text');
        const btnLoader = btn.querySelector('.btn-loader');

        btn.disabled = loading;
        btnText.style.opacity = loading ? '0' : '1';
        btnLoader.style.display = loading ? 'block' : 'none';
    }

    /**
     * Modal management
     */
    showSuccessModal(message) {
        const modal = document.getElementById('successModal');
        const messageElement = document.getElementById('successMessage');
        
        if (modal && messageElement) {
            messageElement.textContent = message;
            modal.style.display = 'flex';
        }
    }

    showErrorModal(message) {
        const modal = document.getElementById('errorModal');
        const messageElement = document.getElementById('errorMessage');
        
        if (modal && messageElement) {
            messageElement.textContent = message;
            modal.style.display = 'flex';
        }
    }

    showDeletePictureModal() {
        const modal = document.getElementById('deletePictureModal');
        if (modal) {
            modal.style.display = 'flex';
        }
    }

    closeModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.style.display = 'none';
        }
    }
}

// Global functions for modal closing (called from HTML)
function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.style.display = 'none';
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new TestimonialEditHandler();
});

// Handle clicks outside modals
document.addEventListener('click', (e) => {
    if (e.target.classList.contains('modal')) {
        e.target.style.display = 'none';
    }
});
