/**
 * Doktor Duzenle (Doctor Edit) Page JavaScript
 * Handles general settings form, picture uploads, CV uploads, and file deletion
 */

class DoktorEditHandler {
    constructor() {
        this.drid = window.pageData.drid;
        this.generalForm = document.getElementById('generalForm');
        this.pictureForm = document.getElementById('pictureForm');
        this.cvForm = document.getElementById('cvForm');
        this.generalSubmitBtn = document.getElementById('generalSubmitBtn');
        this.pictureSubmitBtn = document.getElementById('pictureSubmitBtn');
        this.cvSubmitBtn = document.getElementById('cvSubmitBtn');
        this.currentDeleteTarget = null;
        this.originalValues = {};
        this.codeEditors = {};
        
        this.init();
    }

    init() {
        this.initializeOriginalValues();
        this.setupGeneralForm();
        this.setupPictureForm();
        this.setupCvForm();
        this.setupFileUploads();
        this.setupFormValidation();
        this.setupToggleSwitches();
        this.setupFileDeletion();
        this.setupUrlGeneration();
        this.setupSubeToBransFetching();
        // Setup CodeMirror editors after a delay to ensure CodeMirror is loaded
        setTimeout(() => {
            this.setupCodeEditors();
        }, 100);
    }

    /**
     * Store original values for change detection
     */
    initializeOriginalValues() {
        const photoAltTextInput = document.getElementById('photo_alt_text');
        const photoTitleInput = document.getElementById('photo_title');
        const cvAltTextInput = document.getElementById('cv_alt_text');
        const cvTitleInput = document.getElementById('cv_title');
        
        if (photoAltTextInput) {
            this.originalValues.photoAltText = photoAltTextInput.value;
        }
        if (photoTitleInput) {
            this.originalValues.photoTitle = photoTitleInput.value;
        }
        if (cvAltTextInput) {
            this.originalValues.cvAltText = cvAltTextInput.value;
        }
        if (cvTitleInput) {
            this.originalValues.cvTitle = cvTitleInput.value;
        }
    }

    /**
     * Setup Şube -> Branş fetching (same logic as add page)
     */
    setupSubeToBransFetching() {
        const sidSelect = document.getElementById('sid');
        const bridSelect = document.getElementById('brid');
        if (!sidSelect || !bridSelect) return;

        const replaceBranchOptions = (branches) => {
            const firstOption = bridSelect.querySelector('option');
            while (bridSelect.firstChild) bridSelect.removeChild(bridSelect.firstChild);
            if (firstOption) {
                // Normalize first option label to match templates
                firstOption.textContent = firstOption.textContent && firstOption.textContent.trim() !== '' ? firstOption.textContent : 'Bölüm Seçin';
                firstOption.value = '';
                firstOption.removeAttribute('selected');
                bridSelect.appendChild(firstOption);
            } else {
                const opt = document.createElement('option');
                opt.value = '';
                opt.textContent = 'Bölüm Seçin';
                bridSelect.appendChild(opt);
            }
            branches.forEach(b => {
                const o = document.createElement('option');
                o.value = (b.Brid || b.brid || '').toString();
                o.textContent = b.Name || b.name || '';
                if (o.value && o.textContent) bridSelect.appendChild(o);
            });
        };

        sidSelect.addEventListener('change', async () => {
            const rawSid = sidSelect.value;
            const sid = rawSid && rawSid !== '' ? rawSid : '0';
            try {
                const res = await fetch(`/backend/sube/${encodeURIComponent(sid)}/get-branches`, { 
                    method: 'POST', 
                    headers: { 'Accept': 'application/json' } 
                });
                const data = await res.json();
                if (data && data.status === 200 && Array.isArray(data.data)) {
                    replaceBranchOptions(data.data);
                }
            } catch (error) {
                // Silent error handling
            }
        });
    }

    /**
     * Convert date for backend submission
     */
    convertDateForBackend(dateValue) {
        if (!dateValue || dateValue === '') {
            // Return Go's zero time format for empty dates
            return '0001-01-01T00:00:00Z';
        }
        
        // If it's already in the correct format (YYYY-MM-DD), convert to Go time format
        if (dateValue.match(/^\d{4}-\d{2}-\d{2}$/)) {
            // Convert YYYY-MM-DD to Go time format (2006-01-02T15:04:05Z07:00)
            return dateValue + 'T00:00:00Z';
        }
        
        // If it's in the old format "00:00, 01/01/0001", convert to Go zero time
        if (dateValue.includes('00:00, 01/01/0001')) {
            return '0001-01-01T00:00:00Z';
        }
        
        // Try to parse other date formats
        const date = new Date(dateValue);
        if (!isNaN(date.getTime())) {
            return date.toISOString();
        }
        
        // Log for debugging if needed
        console.log('Date conversion failed for:', dateValue);
        
        // Return Go's zero time format for invalid dates
        return '0001-01-01T00:00:00Z';
    }

    /**
     * Setup CodeMirror editors for HTML, CSS, and JavaScript
     */
    setupCodeEditors() {
        // Wait for CodeMirror to be loaded
        if (typeof CodeMirror === 'undefined') {
            setTimeout(() => this.setupCodeEditors(), 100);
            return;
        }

        // HTML Editor
        const htmlTextarea = document.getElementById('doctor_infos_html');
        if (htmlTextarea) {
            this.codeEditors.html = CodeMirror.fromTextArea(htmlTextarea, {
                mode: 'htmlmixed',
                theme: 'monokai',
                lineNumbers: true,
                autoCloseBrackets: true,
                matchBrackets: true,
                indentUnit: 2,
                tabSize: 2,
                lineWrapping: true,
                extraKeys: {
                    "Ctrl-Space": "autocomplete"
                }
            });
        }

        // CSS Editor
        const cssTextarea = document.getElementById('doctor_infos_css');
        if (cssTextarea) {
            this.codeEditors.css = CodeMirror.fromTextArea(cssTextarea, {
                mode: 'css',
                theme: 'monokai',
                lineNumbers: true,
                autoCloseBrackets: true,
                matchBrackets: true,
                indentUnit: 2,
                tabSize: 2,
                lineWrapping: true,
                extraKeys: {
                    "Ctrl-Space": "autocomplete"
                }
            });
        }

        // JavaScript Editor
        const jsTextarea = document.getElementById('doctor_infos_js');
        if (jsTextarea) {
            this.codeEditors.javascript = CodeMirror.fromTextArea(jsTextarea, {
                mode: 'javascript',
                theme: 'monokai',
                lineNumbers: true,
                autoCloseBrackets: true,
                matchBrackets: true,
                indentUnit: 2,
                tabSize: 2,
                lineWrapping: true,
                extraKeys: {
                    "Ctrl-Space": "autocomplete"
                }
            });
        }

        // Sync CodeMirror content with form data
        Object.keys(this.codeEditors).forEach(key => {
            if (this.codeEditors[key]) {
                this.codeEditors[key].on('change', () => {
                    this.codeEditors[key].save();
                });
            }
        });
    }

    /**
     * Setup general settings form (AJAX JSON submission)
     */
    setupGeneralForm() {
        this.generalForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            // Save CodeMirror content to textareas before submit
            Object.keys(this.codeEditors).forEach(key => {
                if (this.codeEditors[key]) {
                    this.codeEditors[key].save();
                }
            });
            
            if (!this.validateGeneralForm()) {
                return;
            }

            this.setGeneralLoadingState(true);

            try {
                // Collect all form data
                const data = {};
                const allInputs = this.generalForm.querySelectorAll('input, textarea, select');
                
                allInputs.forEach(input => {
                    if (input.type === 'checkbox') {
                        data[input.name] = input.checked;
                    } else if (input.type === 'number') {
                        data[input.name] = parseFloat(input.value) || 0;
                    } else if (input.type === 'date') {
                        // Handle date fields with proper conversion
                        if (input.name === 'birth_date') {
                            data[input.name] = this.convertDateForBackend(input.value);
                        } else {
                            data[input.name] = input.value;
                        }
                    } else if (input.type === 'hidden') {
                        if (input.value === 'true') {
                            data[input.name] = true;
                        } else if (input.value === 'false') {
                            data[input.name] = false;
                        } else if (input.name.includes('old_') && document.querySelector(`[name="${input.name.replace('old_', '')}"]`)?.type === 'number') {
                            data[input.name] = parseFloat(input.value) || 0;
                        } else if (input.name.includes('old_') && document.querySelector(`[name="${input.name.replace('old_', '')}"]`)?.type === 'date') {
                            // Handle old date values - convert to proper format for comparison
                            data[input.name] = this.convertDateForBackend(input.value);
                        } else {
                            data[input.name] = input.value;
                        }
                    } else {
                        data[input.name] = input.value;
                    }
                });

                const response = await fetch(`/backend/doctor/${this.drid}/edit`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (result.status === 201) {
                    this.showSuccessModal('Doktor bilgileri başarıyla güncellendi.');
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

                const response = await fetch(`/backend/doctor/${this.drid}/update-picture`, {
                    method: 'POST',
                    body: formData
                });

                const result = await response.json();

                if (result.status === 201) {
                    this.showSuccessModal('Doktor fotoğrafı başarıyla güncellendi.');
                    // Reload page to show updated picture
                    setTimeout(() => {
                        window.location.reload();
                    }, 2000);
                } else {
                    throw new Error(result.message || 'Fotoğraf güncelleme işlemi başarısız oldu.');
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
     * Setup CV form (AJAX FormData submission)
     */
    setupCvForm() {
        this.cvForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            if (!this.validateCvForm()) {
                return;
            }

            this.setCvLoadingState(true);

            try {
                const formData = new FormData();
                
                // Add all form fields to FormData
                const inputs = this.cvForm.querySelectorAll('input');
                inputs.forEach(input => {
                    if (input.type === 'file') {
                        if (input.files.length > 0) {
                            formData.append(input.name, input.files[0]);
                        }
                    } else {
                        formData.append(input.name, input.value);
                    }
                });

                const response = await fetch(`/backend/doctor/${this.drid}/update-cv`, {
                    method: 'POST',
                    body: formData
                });

                const result = await response.json();

                if (result.status === 201) {
                    this.showSuccessModal('Doktor CV\'si başarıyla güncellendi.');
                    // Reload page to show updated CV
                    setTimeout(() => {
                        window.location.reload();
                    }, 2000);
                } else {
                    throw new Error(result.message || 'CV güncelleme işlemi başarısız oldu.');
                }

            } catch (error) {
                console.error('CV form submission error:', error);
                this.showErrorModal(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
            } finally {
                this.setCvLoadingState(false);
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
        const isImage = input.accept.includes('image');
        const isDocument = input.accept.includes('.pdf') || input.accept.includes('.doc');
        
        // Validate file type
        let allowedTypes = [];
        if (isImage) {
            allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp'];
        } else if (isDocument) {
            allowedTypes = ['application/pdf', 'application/msword', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'];
        }
        
        if (allowedTypes.length > 0 && !allowedTypes.includes(file.type)) {
            const fileTypeText = isImage ? 'PNG, JPG veya WEBP' : 'PDF, DOC veya DOCX';
            this.showErrorModal(`Geçersiz dosya türü. ${fileTypeText} dosyası yükleyin.`);
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
        if (isImage) {
            const reader = new FileReader();
            reader.onload = (e) => {
                previewImage.src = e.target.result;
                content.style.display = 'none';
                preview.style.display = 'flex';
                area.classList.add('has-file');
            };
            reader.readAsDataURL(file);
        } else {
            // For documents, show file name
            const fileNameSpan = preview.querySelector('.file-name');
            if (fileNameSpan) {
                fileNameSpan.textContent = file.name;
            }
            content.style.display = 'none';
            preview.style.display = 'flex';
            area.classList.add('has-file');
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

        // TC Kimlik validation
        const tcKimlikField = document.getElementById('tc_kimlik');
        if (tcKimlikField) {
            tcKimlikField.addEventListener('blur', () => {
                this.validateTcKimlik(tcKimlikField);
            });
        }
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
     * Setup file deletion functionality
     */
    setupFileDeletion() {
        const deleteButtons = document.querySelectorAll('.btn-delete-media');
        const confirmBtn = document.getElementById('confirmDeletePictureBtn');
        
        deleteButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                const drid = btn.dataset.drid;
                const type = btn.dataset.type || 'photo';
                
                this.currentDeleteTarget = { drid: drid, type: type, button: btn };
                this.showDeleteFileModal();
            });
        });

        confirmBtn.addEventListener('click', async () => {
            if (!this.currentDeleteTarget) return;
            
            await this.deleteFile(this.currentDeleteTarget);
        });
    }

    /**
     * Setup URL generation from name
     */
    setupUrlGeneration() {
        const firstNameInput = document.getElementById('first_name');
        const lastNameInput = document.getElementById('last_name');
        const urlInput = document.getElementById('url_name');
        
        if (firstNameInput && lastNameInput && urlInput) {
            const generateUrl = () => {
                if (!urlInput.value || urlInput.dataset.autoGenerated === 'true') {
                    const fullName = `${firstNameInput.value} ${lastNameInput.value}`.trim();
                    const urlFriendly = this.generateUrlFriendlyString(fullName);
                    urlInput.value = urlFriendly;
                    urlInput.dataset.autoGenerated = 'true';
                }
            };
            
            firstNameInput.addEventListener('input', generateUrl);
            lastNameInput.addEventListener('input', generateUrl);
            
            urlInput.addEventListener('input', () => {
                // Mark as manually edited if user types
                const fullName = `${firstNameInput.value} ${lastNameInput.value}`.trim();
                if (urlInput.value !== this.generateUrlFriendlyString(fullName)) {
                    urlInput.dataset.autoGenerated = 'false';
                }
            });
        }
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
     * Delete file
     */
    async deleteFile(target) {
        this.setDeleteFileLoadingState(true);

        try {
            const endpoint = target.type === 'cv' ? 'delete-cv' : 'delete-picture';
            const response = await fetch(`/backend/doctor/${target.drid}/${endpoint}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            const result = await response.json();

            if (result.status === 201) {
                const fileTypeText = target.type === 'cv' ? 'CV' : 'fotoğraf';
                this.showSuccessModal(`Doktor ${fileTypeText} başarıyla silindi.`);
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
                throw new Error(result.message || 'Dosya silme işlemi başarısız oldu.');
            }

        } catch (error) {
            console.error('File deletion error:', error);
            this.showErrorModal(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
        } finally {
            this.setDeleteFileLoadingState(false);
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
     * Validate TC Kimlik field
     */
    validateTcKimlik(field) {
        const tc = field.value.trim();
        if (!tc) return true; // Not required
        
        const tcRegex = /^[0-9]{11}$/;
        const isValid = tcRegex.test(tc);
        
        this.setFieldState(field, isValid, isValid ? '' : 'TC Kimlik No 11 haneli olmalıdır.');
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

        // Validate TC Kimlik
        const tcKimlikField = document.getElementById('tc_kimlik');
        if (tcKimlikField && !this.validateTcKimlik(tcKimlikField)) {
            isValid = false;
        }

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
        const altTextInput = document.getElementById('photo_alt_text');
        const titleInput = document.getElementById('photo_title');
        
        let hasFile = fileInput && fileInput.files.length > 0;
        let hasMetadataChange = false;
        
        // Check if alt text or title has changed
        if (altTextInput && altTextInput.value !== this.originalValues.photoAltText) {
            hasMetadataChange = true;
        }
        if (titleInput && titleInput.value !== this.originalValues.photoTitle) {
            hasMetadataChange = true;
        }
        
        if (!hasFile && !hasMetadataChange) {
            this.showErrorModal('Lütfen bir dosya seçin veya görsel bilgilerini değiştirin.');
            return false;
        }
        
        return true;
    }

    /**
     * Validate CV form
     */
    validateCvForm() {
        // Check if file is selected or metadata has changed
        const fileInput = this.cvForm.querySelector('input[type="file"]');
        const altTextInput = document.getElementById('cv_alt_text');
        const titleInput = document.getElementById('cv_title');
        
        let hasFile = fileInput && fileInput.files.length > 0;
        let hasMetadataChange = false;
        
        // Check if alt text or title has changed
        if (altTextInput && altTextInput.value !== this.originalValues.cvAltText) {
            hasMetadataChange = true;
        }
        if (titleInput && titleInput.value !== this.originalValues.cvTitle) {
            hasMetadataChange = true;
        }
        
        if (!hasFile && !hasMetadataChange) {
            this.showErrorModal('Lütfen bir dosya seçin veya dosya bilgilerini değiştirin.');
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
     * Set loading state for CV form
     */
    setCvLoadingState(loading) {
        if (loading) {
            this.cvSubmitBtn.disabled = true;
            this.cvSubmitBtn.querySelector('.btn-text').style.opacity = '0';
            this.cvSubmitBtn.querySelector('.btn-loader').style.display = 'block';
            this.cvForm.classList.add('form-loading');
        } else {
            this.cvSubmitBtn.disabled = false;
            this.cvSubmitBtn.querySelector('.btn-text').style.opacity = '1';
            this.cvSubmitBtn.querySelector('.btn-loader').style.display = 'none';
            this.cvForm.classList.remove('form-loading');
        }
    }

    /**
     * Set loading state for delete file button
     */
    setDeleteFileLoadingState(loading) {
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
     * Show delete file modal
     */
    showDeleteFileModal() {
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
    new DoktorEditHandler();
    
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
window.DoktorEditHandler = DoktorEditHandler;
