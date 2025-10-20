class FileUploadHandler {
    constructor() {
        this.maxFiles = 10;
        this.selectedFiles = [];
        this.maxFileSize = 0;
        this.uploadArea = document.getElementById('fileUploadArea');
        this.fileInput = document.getElementById('fileInput');
        this.filesPreview = document.getElementById('filesPreview');
        this.filesList = document.getElementById('filesList');
        this.uploadActions = document.getElementById('uploadActions');
        this.filesCount = document.getElementById('filesCount');
        this.totalSize = document.getElementById('totalSize');
        this.uploadBtn = document.getElementById('uploadBtn');
        this.cancelBtn = document.getElementById('cancelBtn');
        this.removeAllBtn = document.getElementById('removeAllBtn');
        this.progressBar = document.getElementById('previewProgress');
        this.progressFill = document.getElementById('progressFill');
        this.progressText = document.getElementById('progressText');
        this.uploadForm = document.getElementById('fileUploadForm');
        this.alertContainer = document.getElementById('alertContainer');
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadMaxFileSize();
        this.setupDragAndDrop();
        this.setupFormSubmission();
    }

    setupEventListeners() {
        // File input change event
        if (this.fileInput) {
            this.fileInput.addEventListener('change', (e) => {
                console.log('File input changed:', e.target.files);
                this.handleFileSelect(e.target.files);
            });
        }

        // Remove all files button
        if (this.removeAllBtn) {
            this.removeAllBtn.addEventListener('click', () => {
                this.removeAllFiles();
            });
        }

        // Cancel button
        if (this.cancelBtn) {
            this.cancelBtn.addEventListener('click', () => {
                this.cancelUpload();
            });
        }

        // Upload button
        if (this.uploadBtn) {
            this.uploadBtn.addEventListener('click', (e) => {
                e.preventDefault();
                this.uploadFiles();
            });
        }
    }

    setupDragAndDrop() {
        if (!this.uploadArea) {
            console.warn('Upload area not found, skipping drag and drop setup');
            return;
        }

        // Prevent default drag behaviors
        ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
            this.uploadArea.addEventListener(eventName, this.preventDefaults, false);
            document.body.addEventListener(eventName, this.preventDefaults, false);
        });

        // Highlight drop area when item is dragged over it
        ['dragenter', 'dragover'].forEach(eventName => {
            this.uploadArea.addEventListener(eventName, () => this.highlight(), false);
        });

        ['dragleave', 'drop'].forEach(eventName => {
            this.uploadArea.addEventListener(eventName, () => this.unhighlight(), false);
        });

        // Handle dropped files
        this.uploadArea.addEventListener('drop', (e) => {
            const dt = e.dataTransfer;
            const files = dt.files;
            console.log('Files dropped:', files);
            this.handleFileSelect(files);
        }, false);
    }

    setupFormSubmission() {
        if (!this.uploadForm) {
            console.warn('Upload form not found, skipping form submission setup');
            return;
        }
        
        this.uploadForm.addEventListener('submit', (e) => {
            e.preventDefault();
            this.uploadFiles();
        });
    }

    preventDefaults(e) {
        e.preventDefault();
        e.stopPropagation();
    }

    highlight() {
        this.uploadArea.classList.add('drag-over');
    }

    unhighlight() {
        this.uploadArea.classList.remove('drag-over');
    }

    loadMaxFileSize() {
        // This would typically come from the server
        // For now, we'll set a default value
        this.maxFileSize = 10 * 1024 * 1024; // 10MB
        const maxFileSizeElement = document.getElementById('maxFileSize');
        if (maxFileSizeElement) {
            maxFileSizeElement.textContent = this.formatFileSize(this.maxFileSize);
        }
    }

    handleFileSelect(files) {
        console.log('Handling file selection:', files);
        
        if (!files || files.length === 0) {
            console.log('No files selected');
            return;
        }

        // Check file count limit
        if (files.length > this.maxFiles) {
            this.showAlert(`Maksimum ${this.maxFiles} dosya seçebilirsiniz.`, 'error');
            return;
        }

        // Validate each file
        const validFiles = [];
        for (let i = 0; i < files.length; i++) {
            const file = files[i];
            console.log(`Processing file ${i + 1}:`, file.name, file.size);
            
            if (file.size > this.maxFileSize) {
                this.showAlert(`"${file.name}" dosyası çok büyük. Maksimum dosya boyutu: ${this.formatFileSize(this.maxFileSize)}`, 'error');
                continue;
            }
            
            validFiles.push(file);
        }

        if (validFiles.length === 0) {
            console.log('No valid files after validation');
            return;
        }

        // Add files to selected files array
        this.selectedFiles = [...this.selectedFiles, ...validFiles];
        
        // Check if we exceed the limit after adding
        if (this.selectedFiles.length > this.maxFiles) {
            this.selectedFiles = this.selectedFiles.slice(0, this.maxFiles);
            this.showAlert(`Maksimum ${this.maxFiles} dosya seçebilirsiniz. Fazla dosyalar kaldırıldı.`, 'warning');
        }

        console.log('Selected files after processing:', this.selectedFiles);
        this.updateFilePreview();
        this.updateUploadActions();
    }

    updateFilePreview() {
        if (!this.filesPreview || !this.filesList) {
            console.warn('File preview elements not found');
            return;
        }

        if (this.selectedFiles.length === 0) {
            this.filesPreview.style.display = 'none';
            return;
        }

        this.filesPreview.style.display = 'block';
        this.filesList.innerHTML = '';

        this.selectedFiles.forEach((file, index) => {
            const fileItem = this.createFilePreviewItem(file, index);
            this.filesList.appendChild(fileItem);
        });
    }

    createFilePreviewItem(file, index) {
        const fileItem = document.createElement('div');
        fileItem.className = 'file-preview-item';
        fileItem.setAttribute('data-file-index', index);

        const fileType = this.getFileType(file.name);
        const fileSize = this.formatFileSize(file.size);

        fileItem.innerHTML = `
            <div class="file-preview-icon ${fileType}">
                <i class="fas fa-file"></i>
            </div>
            <div class="file-preview-info">
                <div class="file-preview-name">${file.name}</div>
                <div class="file-preview-details">
                    <span class="file-preview-size">${fileSize}</span>
                    <span class="file-preview-type">${fileType}</span>
                </div>
            </div>
            <button type="button" class="btn-remove-file" data-file-index="${index}">
                <i class="fas fa-times"></i>
            </button>
        `;

        // Add remove file event listener
        const removeBtn = fileItem.querySelector('.btn-remove-file');
        removeBtn.addEventListener('click', () => {
            this.removeFile(index);
        });

        return fileItem;
    }

    removeFile(index) {
        console.log('Removing file at index:', index);
        this.selectedFiles.splice(index, 1);
        this.updateFilePreview();
        this.updateUploadActions();
    }

    removeAllFiles() {
        console.log('Removing all files');
        this.selectedFiles = [];
        this.updateFilePreview();
        this.updateUploadActions();
        if (this.fileInput) {
            this.fileInput.value = '';
        }
    }

    updateUploadActions() {
        if (!this.uploadActions || !this.filesCount || !this.totalSize) {
            console.warn('Upload actions elements not found');
            return;
        }

        if (this.selectedFiles.length === 0) {
            this.uploadActions.style.display = 'none';
            return;
        }

        this.uploadActions.style.display = 'flex';
        
        const totalSize = this.selectedFiles.reduce((sum, file) => sum + file.size, 0);
        
        this.filesCount.textContent = `${this.selectedFiles.length} dosya seçildi`;
        this.totalSize.textContent = this.formatFileSize(totalSize);
    }

    getFileType(filename) {
        const ext = filename.toLowerCase().split('.').pop();
        
        const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'];
        const videoExts = ['mp4', 'avi', 'mov', 'wmv', 'flv', 'webm'];
        const audioExts = ['mp3', 'wav', 'ogg', 'flac'];
        const documentExts = ['doc', 'docx'];
        const pdfExts = ['pdf'];
        const archiveExts = ['zip', 'rar', '7z', 'tar', 'gz'];
        
        if (imageExts.includes(ext)) return 'image';
        if (videoExts.includes(ext)) return 'video';
        if (audioExts.includes(ext)) return 'audio';
        if (documentExts.includes(ext)) return 'document';
        if (pdfExts.includes(ext)) return 'pdf';
        if (archiveExts.includes(ext)) return 'archive';
        
        return 'other';
    }

    formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    async uploadFiles() {
        if (this.selectedFiles.length === 0) {
            this.showAlert('Lütfen yüklenecek dosya seçin.', 'error');
            return;
        }

        console.log('Starting upload of', this.selectedFiles.length, 'files');
        
        this.setUploadingState(true);
        this.showProgress(0, 'Yükleniyor...');

        try {
            // Upload files one by one
            for (let i = 0; i < this.selectedFiles.length; i++) {
                const file = this.selectedFiles[i];
                console.log(`Uploading file ${i + 1}/${this.selectedFiles.length}:`, file.name);
                
                const progress = ((i + 1) / this.selectedFiles.length) * 100;
                this.showProgress(progress, `Yükleniyor... (${i + 1}/${this.selectedFiles.length})`);

                await this.uploadSingleFile(file);
            }

            this.showProgress(100, 'Tamamlandı!');
            this.showAlert(`${this.selectedFiles.length} dosya başarıyla yüklendi.`, 'success');
            
            // Reset form
            this.removeAllFiles();
            
            // Reload page to show updated file list
            setTimeout(() => {
                window.location.reload();
            }, 1500);

        } catch (error) {
            console.error('Upload error:', error);
            this.showAlert('Dosya yükleme sırasında hata oluştu.', 'error');
        } finally {
            this.setUploadingState(false);
            this.hideProgress();
        }
    }

    async uploadSingleFile(file) {
        const formData = new FormData();
        formData.append('file', file);

        const response = await fetch('/backend/add-file', {
            method: 'POST',
            body: formData
        });

        const result = await response.json();
        
        if (result.status !== 201) {
            throw new Error(result.message || 'Upload failed');
        }

        console.log('File uploaded successfully:', result);
        return result;
    }

    setUploadingState(uploading) {
        if (this.uploadBtn) {
            this.uploadBtn.disabled = uploading;
            if (uploading) {
                const btnText = this.uploadBtn.querySelector('.btn-text');
                const btnLoader = this.uploadBtn.querySelector('.btn-loader');
                if (btnText) btnText.style.display = 'none';
                if (btnLoader) btnLoader.style.display = 'flex';
            } else {
                const btnText = this.uploadBtn.querySelector('.btn-text');
                const btnLoader = this.uploadBtn.querySelector('.btn-loader');
                if (btnText) btnText.style.display = 'inline';
                if (btnLoader) btnLoader.style.display = 'none';
            }
        }
        
        if (this.cancelBtn) {
            this.cancelBtn.disabled = uploading;
        }
        
        if (this.fileInput) {
            this.fileInput.disabled = uploading;
        }
    }

    showProgress(percentage, text) {
        if (this.progressBar) {
            this.progressBar.style.display = 'block';
        }
        if (this.progressFill) {
            this.progressFill.style.width = `${percentage}%`;
        }
        if (this.progressText) {
            this.progressText.textContent = text;
        }
    }

    hideProgress() {
        setTimeout(() => {
            if (this.progressBar) {
                this.progressBar.style.display = 'none';
            }
        }, 1000);
    }

    cancelUpload() {
        this.removeAllFiles();
        this.hideProgress();
    }

    showAlert(message, type = 'info') {
        if (!this.alertContainer) {
            console.error('Alert container not found');
            alert(message); // Fallback to browser alert
            return;
        }

        const alertDiv = document.createElement('div');
        alertDiv.className = `alert alert-${type}`;
        alertDiv.innerHTML = `
            <div class="alert-content">
                <i class="fas fa-${type === 'success' ? 'check-circle' : type === 'error' ? 'exclamation-circle' : type === 'warning' ? 'exclamation-triangle' : 'info-circle'}"></i>
                <span>${message}</span>
            </div>
            <button type="button" class="alert-close" onclick="this.parentElement.remove()">
                <i class="fas fa-times"></i>
            </button>
        `;

        this.alertContainer.appendChild(alertDiv);

        // Auto remove after 5 seconds
        setTimeout(() => {
            if (alertDiv.parentElement) {
                alertDiv.remove();
            }
        }, 5000);
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new FileUploadHandler();
});