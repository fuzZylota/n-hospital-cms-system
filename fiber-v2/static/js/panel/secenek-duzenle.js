/**
 * Secenek Duzenle (Option Edit) Page JavaScript
 * Handles general settings form, media uploads, and media deletion
 */

class OptionEditHandler {
    constructor() {
        this.oid = window.pageData.oid;
        this.generalForm = document.getElementById('generalForm');
        this.mediaForm = document.getElementById('mediaForm');
        this.generalSubmitBtn = document.getElementById('generalSubmitBtn');
        this.mediaSubmitBtn = document.getElementById('mediaSubmitBtn');
        this.currentDeleteTarget = null;
        
        this.init();
    }

    init() {
        this.initializeColorPickers();
        this.setupGeneralForm();
        this.setupMediaForm();
        this.setupFileUploads();
        this.setupColorInputs();
        this.setupFormValidation();
        this.setupToggleSwitches();
        this.setupMediaDeletion();
    }


    /**
     * Convert RGBA string to hex
     */
    rgbaToHex(rgba) {
        const rgbaMatch = rgba.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)(?:,\s*[\d.]+)?\)/);
        if (!rgbaMatch) return '#000000';
        
        const r = parseInt(rgbaMatch[1]);
        const g = parseInt(rgbaMatch[2]);
        const b = parseInt(rgbaMatch[3]);
        
        const toHex = (n) => {
            const hex = n.toString(16);
            return hex.length === 1 ? '0' + hex : hex;
        };
        
        return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
    }

    /**
     * Convert hex string to RGBA
     */
    hexToRgba(hex) {
        if (!hex || !hex.startsWith('#')) return 'rgba(0, 0, 0, 1)';
        
        const r = parseInt(hex.slice(1, 3), 16);
        const g = parseInt(hex.slice(3, 5), 16);
        const b = parseInt(hex.slice(5, 7), 16);
        
        return `rgba(${r}, ${g}, ${b}, 1)`;
    }

    /**
     * Initialize color pickers with correct hex values
     */
    initializeColorPickers() {
        const colorFields = ['primary_color', 'secondary_color', 'accent_color', 'background_color', 'font_color'];
        
        colorFields.forEach(fieldName => {
            const colorPicker = document.getElementById(fieldName);
            const textInput = colorPicker?.parentElement?.querySelector('.color-text');
            
            if (colorPicker && textInput) {
                // If text input has RGBA value, convert to hex for color picker
                if (textInput.value && textInput.value.startsWith('rgba')) {
                    const hexValue = this.rgbaToHex(textInput.value);
                    colorPicker.value = hexValue;
                } else if (textInput.value && textInput.value.startsWith('#')) {
                    // If text input has hex value, use it directly
                    colorPicker.value = textInput.value;
                } else {
                    // Set default values if empty
                    const defaults = {
                        'primary_color': '#2563eb',
                        'secondary_color': '#10b981', 
                        'accent_color': '#f59e0b',
                        'background_color': '#ffffff',
                        'font_color': '#1e293b'
                    };
                    colorPicker.value = defaults[fieldName];
                    textInput.value = defaults[fieldName];
                }
            }
        });
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
                // Collect all form data - dead simple approach
                const data = {};
                const allInputs = this.generalForm.querySelectorAll('input, textarea, select');
                
                allInputs.forEach(input => {
                    if (input.type === 'checkbox') {
                        data[input.name] = input.checked;
                    } else if (input.type === 'color') {
                        // For color inputs, convert hex to RGBA
                        data[input.name] = this.hexToRgba(input.value);
                    } else if(input.type === 'hidden') {
                        if(input.value === 'true') {
                            data[input.name] = true;
                        } else if(input.value === 'false') {
                            data[input.name] = false;
                        } else if (input.name.includes('old_') && document.querySelector(`[name="${input.name.replace('old_', '')}"]`).type === 'number') {
                            data[input.name] = parseInt(input.value);
                        } else {
                            data[input.name] = input.value;
                        }
                    } else if(input.type === 'number') {
                        data[input.name] = parseInt(input.value);
                    } else {
                        data[input.name] = input.value;
                    }
                });

                const response = await fetch(`/backend/option/${this.oid}/edit`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (result.status === 201) {
                    this.showSuccessModal('Genel ayarlar başarıyla güncellendi.');
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
     * Setup media form (AJAX FormData submission)
     */
    setupMediaForm() {
        this.mediaForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            if (!this.validateMediaForm()) {
                return;
            }

            this.setMediaLoadingState(true);

            try {
                const formData = new FormData();
                
                // Add all form fields to FormData
                const inputs = this.mediaForm.querySelectorAll('input');
                inputs.forEach(input => {
                    if (input.type === 'file') {
                        if (input.files.length > 0) {
                            formData.append(input.name, input.files[0]);
                        }
                    } else {
                        formData.append(input.name, input.value);
                    }
                });

                const response = await fetch(`/backend/option/${this.oid}/update-picture`, {
                    method: 'POST',
                    body: formData
                });

                const result = await response.json();

                if (result.status === 201) {
                    this.showSuccessModal('Medya dosyaları başarıyla güncellendi.');
                    // Reload page to show updated media
                    setTimeout(() => {
                        window.location.reload();
                    }, 2000);
                } else {
                    throw new Error(result.message || 'Medya güncelleme işlemi başarısız oldu.');
                }

            } catch (error) {
                console.error('Media form submission error:', error);
                this.showErrorModal(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
            } finally {
                this.setMediaLoadingState(false);
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
        const allowedTypes = input.accept.split(',').map(type => type.trim());
        const isValidType = allowedTypes.some(type => {
            if (type.startsWith('.')) {
                return file.name.toLowerCase().endsWith(type.toLowerCase());
            } else if (type.includes('*')) {
                const mainType = type.split('/')[0];
                return file.type.startsWith(mainType);
            } else {
                return file.type === type;
            }
        });

        if (!isValidType) {
            this.showErrorModal('Geçersiz dosya türü. Lütfen uygun bir dosya seçin.');
            return;
        }

        // Validate file size
        let maxSize = 2 * 1024 * 1024; // Default 2MB
        if (input.id === 'site_favicon_path') {
            maxSize = 1024 * 1024; // 1MB for favicon
        } else if (input.id === 'default_page_media_path') {
            maxSize = 5 * 1024 * 1024; // 5MB for default page
        }
        
        if (file.size > maxSize) {
            const maxSizeMB = maxSize / (1024 * 1024);
            this.showErrorModal(`Dosya boyutu ${maxSizeMB}MB'dan büyük olamaz.`);
            return;
        }

        // Create file list for input
        const dt = new DataTransfer();
        dt.items.add(file);
        input.files = dt.files;

        // Show preview
        if (file.type.startsWith('image/')) {
            const reader = new FileReader();
            reader.onload = (e) => {
                previewImage.src = e.target.result;
                content.style.display = 'none';
                preview.style.display = 'flex';
                area.classList.add('has-file');
            };
            reader.readAsDataURL(file);
        }
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
     * Setup color input synchronization
     */
    setupColorInputs() {
        const colorGroups = [
            { picker: 'primary_color', text: 'primary_color' },
            { picker: 'secondary_color', text: 'secondary_color' },
            { picker: 'accent_color', text: 'accent_color' },
            { picker: 'background_color', text: 'background_color' },
            { picker: 'font_color', text: 'font_color' }
        ];

        colorGroups.forEach(group => {
            const picker = document.getElementById(group.picker);
            if (!picker) return;
            
            const textInput = picker.parentNode.querySelector('.color-text');
            if (!textInput) return;

            // Convert hex to rgba
            const hexToRgba = (hex, alpha = 1) => {
                if (!hex || hex.length < 7) return hex;
                const r = parseInt(hex.slice(1, 3), 16);
                const g = parseInt(hex.slice(3, 5), 16);
                const b = parseInt(hex.slice(5, 7), 16);
                return `rgba(${r}, ${g}, ${b}, ${alpha})`;
            };

            // Convert rgba to hex
            const rgbaToHex = (rgba) => {
                if (!rgba) return '#000000';
                const match = rgba.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
                if (match) {
                    const r = parseInt(match[1]).toString(16).padStart(2, '0');
                    const g = parseInt(match[2]).toString(16).padStart(2, '0');
                    const b = parseInt(match[3]).toString(16).padStart(2, '0');
                    return `#${r}${g}${b}`;
                }
                return rgba.startsWith('#') ? rgba : '#000000';
            };

            // Initialize text input with rgba value from picker
            const initialHex = picker.value;
            if (initialHex && initialHex.startsWith('#')) {
                textInput.value = hexToRgba(initialHex);
            } else if (initialHex && initialHex.startsWith('rgba')) {
                // If picker has RGBA value, convert to hex first
                const hexValue = this.rgbaToHex(initialHex);
                picker.value = hexValue;
                textInput.value = hexToRgba(hexValue);
            } else {
                // If picker doesn't have a value, set default based on field name
                let defaultColor = '#000000';
                switch (group.picker) {
                    case 'primary_color': defaultColor = '#2563eb'; break;
                    case 'secondary_color': defaultColor = '#10b981'; break;
                    case 'accent_color': defaultColor = '#f59e0b'; break;
                    case 'background_color': defaultColor = '#ffffff'; break;
                    case 'font_color': defaultColor = '#1e293b'; break;
                }
                picker.value = defaultColor;
                textInput.value = hexToRgba(defaultColor);
            }

            // Update text when picker changes
            picker.addEventListener('input', (e) => {
                const rgbaValue = hexToRgba(e.target.value);
                textInput.value = rgbaValue;
            });

            // Update picker when text changes
            textInput.addEventListener('input', (e) => {
                const hexValue = rgbaToHex(e.target.value);
                picker.value = hexValue;
            });
        });
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

        // URL validation
        const urlFields = this.generalForm.querySelectorAll('input[type="url"]');
        urlFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validateUrl(field);
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
     * Setup media deletion functionality
     */
    setupMediaDeletion() {
        const deleteButtons = document.querySelectorAll('.btn-delete-media');
        const confirmBtn = document.getElementById('confirmDeleteMediaBtn');
        
        deleteButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                const mediaType = btn.dataset.type;
                const oid = btn.dataset.oid;
                
                this.currentDeleteTarget = { type: mediaType, oid: oid, button: btn };
                this.showDeleteMediaModal();
            });
        });

        confirmBtn.addEventListener('click', async () => {
            if (!this.currentDeleteTarget) return;
            
            await this.deleteMedia(this.currentDeleteTarget);
        });
    }

    /**
     * Delete media file
     */
    async deleteMedia(target) {
        this.setDeleteMediaLoadingState(true);

        try {
            const response = await fetch(`/backend/option/${target.oid}/delete-picture`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    media_type: target.type
                })
            });

            const result = await response.json();

            if (result.status === 201) {
                this.showSuccessModal('Medya dosyası başarıyla silindi.');
                // Remove the media item from UI
                const mediaItem = target.button.closest('.media-item');
                if (mediaItem) {
                    mediaItem.classList.add('deleting');
                    setTimeout(() => {
                        window.location.reload();
                    }, 1000);
                }
                this.closeModal('deleteMediaModal');
            } else {
                throw new Error(result.message || 'Medya silme işlemi başarısız oldu.');
            }

        } catch (error) {
            console.error('Media deletion error:', error);
            this.showErrorModal(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
        } finally {
            this.setDeleteMediaLoadingState(false);
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
     * Validate URL field
     */
    validateUrl(field) {
        const url = field.value.trim();
        if (!url) return true; // Not required
        
        try {
            new URL(url);
            this.setFieldState(field, true, '');
            return true;
        } catch {
            this.setFieldState(field, false, 'Geçerli bir URL girin.');
            return false;
        }
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

        // Validate URL fields
        const urlFields = this.generalForm.querySelectorAll('input[type="url"]');
        urlFields.forEach(field => {
            if(field.value.trim() === '' || field.value.trim() === "#") return;
            if (!this.validateUrl(field)) {
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
     * Validate media form
     */
    validateMediaForm() {
        // Check if at least one file is selected or meta data is provided
        const fileInputs = this.mediaForm.querySelectorAll('input[type="file"]');
        const textInputs = this.mediaForm.querySelectorAll('input[type="text"]');
        
        let hasFileOrText = false;
        
        fileInputs.forEach(input => {
            if (input.files.length > 0) {
                hasFileOrText = true;
            }
        });
        
        textInputs.forEach(input => {
            if (input.value.trim()) {
                hasFileOrText = true;
            }
        });
        
        if (!hasFileOrText) {
            this.showErrorModal('Lütfen en az bir dosya seçin veya metin bilgisi girin.');
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
     * Set loading state for media form
     */
    setMediaLoadingState(loading) {
        if (loading) {
            this.mediaSubmitBtn.disabled = true;
            this.mediaSubmitBtn.querySelector('.btn-text').style.opacity = '0';
            this.mediaSubmitBtn.querySelector('.btn-loader').style.display = 'block';
            this.mediaForm.classList.add('form-loading');
        } else {
            this.mediaSubmitBtn.disabled = false;
            this.mediaSubmitBtn.querySelector('.btn-text').style.opacity = '1';
            this.mediaSubmitBtn.querySelector('.btn-loader').style.display = 'none';
            this.mediaForm.classList.remove('form-loading');
        }
    }

    /**
     * Set loading state for delete media button
     */
    setDeleteMediaLoadingState(loading) {
        const confirmBtn = document.getElementById('confirmDeleteMediaBtn');
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
     * Show delete media modal
     */
    showDeleteMediaModal() {
        const modal = document.getElementById('deleteMediaModal');
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
            if (modalId === 'deleteMediaModal') {
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
    new OptionEditHandler();
    
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
window.OptionEditHandler = OptionEditHandler;
