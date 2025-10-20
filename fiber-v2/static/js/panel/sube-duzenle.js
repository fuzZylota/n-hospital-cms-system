/**
 * Sube Duzenle (Branch Edit) Page JavaScript
 * Handles general settings form, picture uploads, and picture deletion
 */

class SubeEditHandler {
    constructor() {
        this.sid = window.pageData.sid;
        this.generalForm = document.getElementById('generalForm');
        this.pictureForm = document.getElementById('pictureForm');
        this.documentForm = document.getElementById('documentForm');
        this.generalSubmitBtn = document.getElementById('generalSubmitBtn');
        this.pictureSubmitBtn = document.getElementById('pictureSubmitBtn');
        this.documentUploadBtn = document.getElementById('documentUploadBtn');
        this.currentDeleteTarget = null;
        this.currentDeleteDocumentTarget = null;
        this.currentEditDocumentTarget = null;
        this.originalValues = {};
        this.selectedFiles = [];
        
        this.init();
    }

    init() {
        this.initializeOriginalValues();
        this.setupGeneralForm();
        this.setupPictureForm();
        this.setupDocumentForm();
        this.setupFileUploads();
        this.setupDocumentUploads();
        this.setupFormValidation();
        this.setupToggleSwitches();
        this.setupPictureDeletion();
        this.setupDocumentDeletion();
        this.setupDocumentEditing();
        this.setupUrlGeneration();
    }

    /**
     * Store original values for change detection
     */
    initializeOriginalValues() {
        const altTextInput = document.getElementById('sube_media_alt_text');
        const titleInput = document.getElementById('sube_media_title');
        
        if (altTextInput) {
            this.originalValues.altText = altTextInput.value;
        }
        if (titleInput) {
            this.originalValues.title = titleInput.value;
        }
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
                        } else {
                            data[input.name] = input.value;
                        }
                    } else {
                        data[input.name] = input.value;
                    }
                });

                const response = await fetch(`/backend/sube/${this.sid}/edit`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (result.status === 201) {
                    this.showSuccessModal('Şube bilgileri başarıyla güncellendi.');
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
     * Setup document form (AJAX FormData submission)
     */
    setupDocumentForm() {
        this.documentForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            if (this.selectedFiles.length === 0) {
                this.showErrorModal('Lütfen en az bir dosya seçin.');
                return;
            }

            this.setDocumentLoadingState(true);

            try {
                const formData = new FormData();
                
                // Add files and their data
                this.selectedFiles.forEach((fileData, index) => {
                    formData.append(`file${index + 1}`, fileData.file);
                    formData.append(`data${index + 1}`, fileData.data || '');
                });

                const response = await fetch(`/backend/sube/${this.sid}/add-documents`, {
                    method: 'POST',
                    body: formData
                });

                const result = await response.json();

                if (result.status === 201) {
                    this.showSuccessModal('Dokümanlar başarıyla yüklendi.');
                    // Clear the form
                    this.clearDocumentForm();
                } else {
                    throw new Error(result.message || 'Doküman yükleme işlemi başarısız oldu.');
                }

            } catch (error) {
                console.error('Document form submission error:', error);
                this.showErrorModal(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
            } finally {
                this.setDocumentLoadingState(false);
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

                const response = await fetch(`/backend/sube/${this.sid}/update-picture`, {
                    method: 'POST',
                    body: formData
                });

                const result = await response.json();

                if (result.status === 201) {
                    this.showSuccessModal('Şube görseli başarıyla güncellendi.');
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
     * Setup document upload areas with drag & drop functionality
     */
    setupDocumentUploads() {
        const documentUploadArea = document.getElementById('documentUploadArea');
        const documentInput = document.getElementById('documentInput');
        const documentPreview = document.getElementById('documentPreview');
        const documentFilesList = document.getElementById('documentFilesList');
        const documentUploadActions = document.getElementById('documentUploadActions');
        const documentFilesCount = document.getElementById('documentFilesCount');
        const documentTotalSize = document.getElementById('documentTotalSize');
        const removeAllDocumentsBtn = document.getElementById('removeAllDocumentsBtn');
        const documentCancelBtn = document.getElementById('documentCancelBtn');

        // Click to browse
        documentUploadArea.addEventListener('click', (e) => {
            if (e.target === documentUploadArea || e.target.closest('.upload-content')) {
                documentInput.click();
            }
        });

        // Drag & drop events
        documentUploadArea.addEventListener('dragover', (e) => {
            e.preventDefault();
            documentUploadArea.classList.add('dragover');
        });

        documentUploadArea.addEventListener('dragleave', (e) => {
            e.preventDefault();
            if (!documentUploadArea.contains(e.relatedTarget)) {
                documentUploadArea.classList.remove('dragover');
            }
        });

        documentUploadArea.addEventListener('drop', (e) => {
            e.preventDefault();
            documentUploadArea.classList.remove('dragover');
            
            const files = Array.from(e.dataTransfer.files);
            this.handleDocumentFiles(files);
        });

        // File input change
        documentInput.addEventListener('change', (e) => {
            const files = Array.from(e.target.files);
            this.handleDocumentFiles(files);
        });

        // Remove all files
        removeAllDocumentsBtn.addEventListener('click', () => {
            this.clearDocumentForm();
        });

        // Cancel upload
        documentCancelBtn.addEventListener('click', () => {
            this.clearDocumentForm();
        });
    }

    /**
     * Handle document files selection
     */
    handleDocumentFiles(files) {
        if (files.length === 0) return;

        // Limit to 10 files
        const limitedFiles = files.slice(0, 10);
        
        // Add files to selectedFiles array
        limitedFiles.forEach(file => {
            this.selectedFiles.push({
                file: file,
                data: '',
                id: Date.now() + Math.random()
            });
        });

        this.updateDocumentPreview();
        this.updateDocumentActions();
    }

    /**
     * Update document preview
     */
    updateDocumentPreview() {
        const documentPreview = document.getElementById('documentPreview');
        const documentFilesList = document.getElementById('documentFilesList');
        
        if (this.selectedFiles.length === 0) {
            documentPreview.style.display = 'none';
            return;
        }

        documentPreview.style.display = 'block';
        documentFilesList.innerHTML = '';

        this.selectedFiles.forEach((fileData, index) => {
            const fileItem = document.createElement('div');
            fileItem.className = 'file-item';
            fileItem.innerHTML = `
                <div class="file-info">
                    <div class="file-icon">
                        <i class="fas fa-file"></i>
                    </div>
                    <div class="file-details">
                        <div class="file-name">${fileData.file.name}</div>
                        <div class="file-size">${this.formatBytes(fileData.file.size)}</div>
                    </div>
                    <button type="button" class="btn-remove-file" data-id="${fileData.id}">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
                <div class="file-data-input">
                    <input type="text" class="form-input" placeholder="Dosya için ek bilgi (opsiyonel)" 
                           value="${fileData.data}" data-id="${fileData.id}">
                </div>
            `;
            documentFilesList.appendChild(fileItem);
        });

        // Add event listeners for remove buttons and data inputs
        documentFilesList.querySelectorAll('.btn-remove-file').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const id = e.target.closest('.btn-remove-file').dataset.id;
                this.removeDocumentFile(id);
            });
        });

        documentFilesList.querySelectorAll('.file-data-input input').forEach(input => {
            input.addEventListener('input', (e) => {
                const id = e.target.dataset.id;
                const fileData = this.selectedFiles.find(f => f.id == id);
                if (fileData) {
                    fileData.data = e.target.value;
                }
            });
        });
    }

    /**
     * Remove a document file
     */
    removeDocumentFile(id) {
        this.selectedFiles = this.selectedFiles.filter(f => f.id != id);
        this.updateDocumentPreview();
        this.updateDocumentActions();
    }

    /**
     * Update document actions visibility and info
     */
    updateDocumentActions() {
        const documentUploadActions = document.getElementById('documentUploadActions');
        const documentFilesCount = document.getElementById('documentFilesCount');
        const documentTotalSize = document.getElementById('documentTotalSize');

        if (this.selectedFiles.length === 0) {
            documentUploadActions.style.display = 'none';
            return;
        }

        documentUploadActions.style.display = 'flex';
        documentFilesCount.textContent = `${this.selectedFiles.length} dosya seçildi`;
        
        const totalSize = this.selectedFiles.reduce((sum, fileData) => sum + fileData.file.size, 0);
        documentTotalSize.textContent = this.formatBytes(totalSize);
    }

    /**
     * Clear document form
     */
    clearDocumentForm() {
        this.selectedFiles = [];
        document.getElementById('documentInput').value = '';
        document.getElementById('documentPreview').style.display = 'none';
        document.getElementById('documentUploadActions').style.display = 'none';
    }

    /**
     * Format bytes to human readable format
     */
    formatBytes(bytes, decimals = 2) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const dm = decimals < 0 ? 0 : decimals;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
    }

    /**
     * Setup file upload areas with drag & drop functionality
     */
    setupFileUploads() {
        const fileUploadAreas = document.querySelectorAll('.file-upload-area:not(#documentUploadArea)');
        
        fileUploadAreas.forEach(area => {
            const input = area.querySelector('.file-input');
            const content = area.querySelector('.file-upload-content');
            const preview = area.querySelector('.file-preview');
            const previewImage = area.querySelector('.preview-image');
            const removeBtn = area.querySelector('.file-remove');
            const browseLink = area.querySelector('.file-browse');

            // Skip if required elements don't exist
            if (!input || !content || !preview || !previewImage || !removeBtn || !browseLink) {
                return;
            }

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

        // URL validation
        const urlFields = this.generalForm.querySelectorAll('input[type="url"]');
        urlFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validateUrl(field);
            });
        });

        // Phone validation
        const phoneFields = this.generalForm.querySelectorAll('input[type="tel"]');
        phoneFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validatePhone(field);
            });
        });

        // Coordinate validation
        const latitudeField = document.getElementById('latitude');
        const longitudeField = document.getElementById('longitude');
        
        if (latitudeField) {
            latitudeField.addEventListener('blur', () => {
                this.validateLatitude(latitudeField);
            });
        }
        
        if (longitudeField) {
            longitudeField.addEventListener('blur', () => {
                this.validateLongitude(longitudeField);
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
     * Setup document deletion functionality
     */
    setupDocumentDeletion() {
        const deleteButtons = document.querySelectorAll('.btn-delete-document');
        const confirmBtn = document.getElementById('confirmDeleteDocumentBtn');
        
        deleteButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                const documentMid = btn.dataset.mid;
                const documentItem = btn.closest('.document-item');
                
                this.currentDeleteDocumentTarget = { 
                    mid: documentMid, 
                    button: btn, 
                    item: documentItem 
                };
                this.showDeleteDocumentModal();
            });
        });

        confirmBtn.addEventListener('click', async () => {
            if (!this.currentDeleteDocumentTarget) return;
            
            await this.deleteDocument(this.currentDeleteDocumentTarget);
        });
    }

    /**
     * Delete document
     */
    async deleteDocument(target) {
        this.setDeleteDocumentLoadingState(true);

        try {
            const response = await fetch(`/backend/sube/${this.sid}/delete-document`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    document_mid: target.mid
                })
            });

            const result = await response.json();

            if (result.status === 201) {
                this.showSuccessModal('Doküman başarıyla silindi.');
                // Remove the document item from UI
                if (target.item) {
                    target.item.classList.add('deleting');
                    setTimeout(() => {
                        target.item.remove();
                        // Check if no documents left
                        const documentsGrid = document.querySelector('.documents-grid');
                        if (documentsGrid && documentsGrid.children.length === 0) {
                            const noDocuments = document.querySelector('.no-documents');
                            if (!noDocuments) {
                                documentsGrid.parentNode.innerHTML = `
                                    <div class="no-documents">
                                        <i class="fas fa-file-alt"></i>
                                        <span>Henüz doküman yüklenmemiş</span>
                                    </div>
                                `;
                            }
                        }
                    }, 1000);
                }
                this.closeModal('deleteDocumentModal');
            } else {
                throw new Error(result.message || 'Doküman silme işlemi başarısız oldu.');
            }

        } catch (error) {
            console.error('Document deletion error:', error);
            this.showErrorModal(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
        } finally {
            this.setDeleteDocumentLoadingState(false);
        }
    }

    /**
     * Setup document editing functionality
     */
    setupDocumentEditing() {
        const editButtons = document.querySelectorAll('.btn-edit-document');
        const confirmBtn = document.getElementById('confirmEditDocumentBtn');
        const editForm = document.getElementById('editDocumentForm');
        const editDataInput = document.getElementById('editDocumentData');
        
        editButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                const documentMid = btn.dataset.mid;
                const documentItem = btn.closest('.document-item');
                const currentData = documentItem.querySelector('.document-data')?.textContent?.replace('Ek Bilgi: ', '') || '';
                
                this.currentEditDocumentTarget = { 
                    mid: documentMid, 
                    button: btn, 
                    item: documentItem 
                };
                
                // Set current data in the form
                editDataInput.value = currentData;
                this.showEditDocumentModal();
            });
        });

        confirmBtn.addEventListener('click', async () => {
            if (!this.currentEditDocumentTarget) return;
            
            const newData = editDataInput.value.trim();
            await this.editDocument(this.currentEditDocumentTarget, newData);
        });
    }

    /**
     * Edit document data
     */
    async editDocument(target, newData) {
        this.setEditDocumentLoadingState(true);

        try {
            const response = await fetch(`/backend/sube/${this.sid}/edit-document`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    document_mid: target.mid,
                    data: newData
                })
            });

            const result = await response.json();

            if (result.status === 201) {
                this.showSuccessModal('Doküman bilgileri başarıyla güncellendi.');
                // Update the document item in UI
                if (target.item) {
                    const documentDataElement = target.item.querySelector('.document-data');
                    if (newData) {
                        if (documentDataElement) {
                            documentDataElement.innerHTML = `<strong>Ek Bilgi:</strong> ${newData}`;
                        } else {
                            // Create new data element if it doesn't exist
                            const documentInfo = target.item.querySelector('.document-info');
                            const dataElement = document.createElement('div');
                            dataElement.className = 'document-data';
                            dataElement.innerHTML = `<strong>Ek Bilgi:</strong> ${newData}`;
                            documentInfo.appendChild(dataElement);
                        }
                    } else {
                        // Remove data element if newData is empty
                        if (documentDataElement) {
                            documentDataElement.remove();
                        }
                    }
                }
                this.closeModal('editDocumentModal');
            } else {
                throw new Error(result.message || 'Doküman güncelleme işlemi başarısız oldu.');
            }

        } catch (error) {
            console.error('Document editing error:', error);
            this.showErrorModal(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
        } finally {
            this.setEditDocumentLoadingState(false);
        }
    }

    /**
     * Setup picture deletion functionality
     */
    setupPictureDeletion() {
        const deleteButtons = document.querySelectorAll('.btn-delete-media');
        const confirmBtn = document.getElementById('confirmDeletePictureBtn');
        
        deleteButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                const sid = btn.dataset.sid;
                
                this.currentDeleteTarget = { sid: sid, button: btn };
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
            const response = await fetch(`/backend/sube/${target.sid}/delete-picture`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            const result = await response.json();

            if (result.status === 201) {
                this.showSuccessModal('Şube görseli başarıyla silindi.');
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
     * Validate URL field
     */
    validateUrl(field) {
        const url = field.value.trim();
        if (!url) return true; // Not required

        // Accept without protocol and require at least one dot (e.g., example.com)
        // Optional protocol, no spaces, domain with dots, optional path/query
        const domainLikeRegex = /^(?:[a-zA-Z]+:\/\/)?[^\s.]+(?:\.[^\s.]+)+(?:[\/#?].*)?$/;
        const isValid = domainLikeRegex.test(url);
        this.setFieldState(field, isValid, isValid ? '' : "Geçerli bir URL girin (örn: ornek.com)");
        return isValid;
    }

    /**
     * Validate phone field
     */
    validatePhone(field) {
        const phone = field.value.trim();
        if (!phone) return true; // Not required
        
        const phoneRegex = /^[\+]?[0-9\s\-\(\)]{7,}$/;
        const isValid = phoneRegex.test(phone);
        
        this.setFieldState(field, isValid, isValid ? '' : 'Geçerli bir telefon numarası girin.');
        return isValid;
    }

    /**
     * Validate latitude field
     */
    validateLatitude(field) {
        const lat = parseFloat(field.value);
        if (isNaN(lat)) return true; // Empty is ok
        
        const isValid = lat >= -90 && lat <= 90;
        this.setFieldState(field, isValid, isValid ? '' : 'Enlem -90 ile 90 arasında olmalıdır.');
        return isValid;
    }

    /**
     * Validate longitude field
     */
    validateLongitude(field) {
        const lng = parseFloat(field.value);
        if (isNaN(lng)) return true; // Empty is ok
        
        const isValid = lng >= -180 && lng <= 180;
        this.setFieldState(field, isValid, isValid ? '' : 'Boylam -180 ile 180 arasında olmalıdır.');
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

        // Validate URL fields
        const urlFields = this.generalForm.querySelectorAll('input[type="url"]');
        urlFields.forEach(field => {
            if (field.value.trim() && !this.validateUrl(field)) {
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

        // Validate coordinates
        const latField = document.getElementById('latitude');
        const lngField = document.getElementById('longitude');
        if (latField && !this.validateLatitude(latField)) isValid = false;
        if (lngField && !this.validateLongitude(lngField)) isValid = false;

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
        const altTextInput = document.getElementById('sube_media_alt_text');
        const titleInput = document.getElementById('sube_media_title');
        
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
     * Set loading state for document form
     */
    setDocumentLoadingState(loading) {
        if (loading) {
            this.documentUploadBtn.disabled = true;
            this.documentUploadBtn.querySelector('.btn-text').style.opacity = '0';
            this.documentUploadBtn.querySelector('.btn-loader').style.display = 'block';
            this.documentForm.classList.add('form-loading');
        } else {
            this.documentUploadBtn.disabled = false;
            this.documentUploadBtn.querySelector('.btn-text').style.opacity = '1';
            this.documentUploadBtn.querySelector('.btn-loader').style.display = 'none';
            this.documentForm.classList.remove('form-loading');
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
     * Set loading state for delete document button
     */
    setDeleteDocumentLoadingState(loading) {
        const confirmBtn = document.getElementById('confirmDeleteDocumentBtn');
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
     * Set loading state for edit document button
     */
    setEditDocumentLoadingState(loading) {
        const confirmBtn = document.getElementById('confirmEditDocumentBtn');
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
     * Show delete document modal
     */
    showDeleteDocumentModal() {
        const modal = document.getElementById('deleteDocumentModal');
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }

    /**
     * Show edit document modal
     */
    showEditDocumentModal() {
        const modal = document.getElementById('editDocumentModal');
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
    new SubeEditHandler();
    
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
window.SubeEditHandler = SubeEditHandler;
