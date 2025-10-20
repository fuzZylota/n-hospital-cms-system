/**
 * Dosyalar (File Manager) Page JavaScript
 * Handles file listing, preview, management, and deletion
 */

class FileManager {
    constructor() {
        this.filesData = window.filesData || {};
        this.files = [];
        this.filteredFiles = [];
        this.currentView = 'grid';
        this.currentFile = null;
        
        this.searchInput = document.getElementById('searchInput');
        this.typeFilter = document.getElementById('typeFilter');
        this.sortFilter = document.getElementById('sortFilter');
        this.refreshBtn = document.getElementById('refreshBtn');
        this.filesContainer = document.getElementById('filesContainer');
        this.filesCount = document.getElementById('filesCount');
        this.totalSize = document.getElementById('totalSize');
        this.viewBtns = document.querySelectorAll('.view-btn');
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupViewControls();
        this.setupModals();
        this.setupFileItemListeners();
        this.loadFiles();
        this.formatFileSizes();
    }

    /**
     * Setup all event listeners
     */
    setupEventListeners() {
        // Search input
        this.searchInput.addEventListener('input', () => {
            this.filterFiles();
        });

        // Type filter
        this.typeFilter.addEventListener('change', () => {
            this.filterFiles();
        });

        // Sort filter
        this.sortFilter.addEventListener('change', () => {
            this.sortFiles();
        });

        // Refresh button
        this.refreshBtn.addEventListener('click', () => {
            this.loadFiles();
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.closeModals();
            }
        });
    }

    /**
     * Setup view controls
     */
    setupViewControls() {
        this.viewBtns.forEach(btn => {
            btn.addEventListener('click', () => {
                const view = btn.getAttribute('data-view');
                this.setView(view);
            });
        });
    }

    /**
     * Setup modals
     */
    setupModals() {
        // Close modals when clicking outside
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) {
                this.closeModals();
            }
        });
    }

    /**
     * Load files from server-side data
     */
    loadFiles() {
        // Get files from server-side data
        this.files = this.filesData.files || [];
        this.filteredFiles = [...this.files];
        this.updateStats();
    }

    /**
     * Filter files based on search and type
     */
    filterFiles() {
        const searchTerm = this.searchInput.value.toLowerCase();
        const typeFilter = this.typeFilter.value;
        
        // Get all file items from DOM
        const fileItems = this.filesContainer.querySelectorAll('.file-item');
        
        fileItems.forEach(item => {
            const fileName = item.getAttribute('data-file-name');
            const file = this.files.find(f => f.name === fileName);
            
            if (!file) return;
            
            const matchesSearch = file.name.toLowerCase().includes(searchTerm);
            const matchesType = !typeFilter || file.type === typeFilter;
            const shouldShow = matchesSearch && matchesType;
            
            // Show/hide file item
            item.style.display = shouldShow ? 'flex' : 'none';
        });
        
        this.updateStats();
    }

    /**
     * Sort files based on selected criteria
     */
    sortFiles() {
        const sortBy = this.sortFilter.value;
        const filesGrid = this.filesContainer.querySelector('#filesGrid');
        
        if (!filesGrid) return;
        
        // Get all visible file items
        const fileItems = Array.from(filesGrid.querySelectorAll('.file-item'))
            .filter(item => item.style.display !== 'none');
        
        // Sort file items based on criteria
        fileItems.sort((a, b) => {
            const fileNameA = a.getAttribute('data-file-name');
            const fileNameB = b.getAttribute('data-file-name');
            const fileA = this.files.find(f => f.name === fileNameA);
            const fileB = this.files.find(f => f.name === fileNameB);
            
            if (!fileA || !fileB) return 0;
            
            switch (sortBy) {
                case 'name':
                    return fileA.name.localeCompare(fileB.name);
                case 'size':
                    return fileB.size - fileA.size;
                case 'date':
                    // Since we don't have date in the current data structure, sort by name
                    return fileA.name.localeCompare(fileB.name);
                default:
                    return 0;
            }
        });
        
        // Reorder DOM elements
        fileItems.forEach(item => {
            filesGrid.appendChild(item);
        });
    }

    /**
     * Set view mode (grid or list)
     */
    setView(view) {
        this.currentView = view;
        
        // Update view buttons
        this.viewBtns.forEach(btn => {
            btn.classList.toggle('active', btn.getAttribute('data-view') === view);
        });
        
        // Update container class
        this.filesContainer.className = `files-container ${view === 'grid' ? 'files-grid' : 'files-list'}`;
        
        // Update files grid class
        const filesGrid = this.filesContainer.querySelector('#filesGrid');
        if (filesGrid) {
            filesGrid.className = view === 'grid' ? 'files-grid' : 'files-list';
        }
    }

    /**
     * Format file sizes in the template
     */
    formatFileSizes() {
        const fileSizeElements = document.querySelectorAll('.file-size[data-size]');
        fileSizeElements.forEach(element => {
            const sizeInBytes = parseInt(element.getAttribute('data-size'));
            element.textContent = this.formatFileSize(sizeInBytes);
        });
    }

    /**
     * Setup file item event listeners
     */
    setupFileItemListeners() {
        // Add event listeners to file items
        this.filesContainer.querySelectorAll('.file-item').forEach(item => {
            item.addEventListener('click', (e) => {
                if (!e.target.closest('.file-actions')) {
                    const fileName = item.getAttribute('data-file-name');
                    this.previewFile(fileName);
                }
            });
        });
        
        // Add event listeners to action buttons
        this.filesContainer.querySelectorAll('.action-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.stopPropagation();
                const action = btn.getAttribute('data-action');
                const fileName = btn.getAttribute('data-file-name');
                this.handleFileAction(action, fileName);
            });
        });
    }

    /**
     * Create file item HTML
     */
    createFileItem(file) {
        const fileType = this.getFileType(file.name);
        const fileIcon = this.getFileIcon(fileType);
        const fileSize = this.formatFileSize(file.size);
        const fileDate = this.formatDate(file.date);
        
        return `
            <div class="file-item" data-file-name="${file.name}">
                <div class="file-icon ${fileType}">
                    <i class="${fileIcon}"></i>
                </div>
                <div class="file-info">
                    <div class="file-name">${file.name}</div>
                    <div class="file-details">
                        <span class="file-size">${fileSize}</span>
                        <span class="file-date">${fileDate}</span>
                    </div>
                </div>
                <div class="file-actions">
                    <button class="action-btn preview" data-action="preview" data-file-name="${file.name}" title="Önizle">
                        <i class="fas fa-eye"></i>
                    </button>
                    <button class="action-btn download" data-action="download" data-file-name="${file.name}" title="İndir">
                        <i class="fas fa-download"></i>
                    </button>
                    <button class="action-btn delete" data-action="delete" data-file-name="${file.name}" title="Sil">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            </div>
        `;
    }

    /**
     * Handle file actions
     */
    handleFileAction(action, fileName) {
        switch (action) {
            case 'preview':
                this.previewFile(fileName);
                break;
            case 'download':
                this.downloadFile(fileName);
                break;
            case 'delete':
                this.deleteFile(fileName);
                break;
        }
    }

    /**
     * Preview file in modal
     */
    previewFile(fileName) {
        const file = this.files.find(f => f.name === fileName);
        if (!file) return;
        
        this.currentFile = file;
        
        const modal = document.getElementById('filePreviewModal');
        const previewContent = document.getElementById('previewContent');
        const previewFileName = document.getElementById('previewFileName');
        const previewFileInfo = document.getElementById('previewFileInfo');
        
        previewFileName.textContent = file.name;
        previewFileInfo.innerHTML = `
            <div class="file-info">
                <div><strong>Boyut:</strong> ${this.formatFileSize(file.size)}</div>
                <div><strong>Tarih:</strong> ${this.formatDate(file.date)}</div>
                <div><strong>Tür:</strong> ${file.type}</div>
            </div>
        `;
        
        // Set preview content based on file type
        const fileType = this.getFileType(file.name);
        const fileUrl = this.filesData.uploadsPath + file.name;
        
        switch (fileType) {
            case 'image':
                this.showImagePreview(fileUrl, file.name);
                break;
            case 'video':
                previewContent.innerHTML = `<video controls><source src="${fileUrl}" type="video/mp4"></video>`;
                document.getElementById('imageZoomControls').style.display = 'none';
                break;
            case 'audio':
                previewContent.innerHTML = `<audio controls><source src="${fileUrl}" type="audio/mpeg"></audio>`;
                document.getElementById('imageZoomControls').style.display = 'none';
                break;
            case 'document':
                // Check if it's a PDF specifically
                if (file.name.toLowerCase().endsWith('.pdf')) {
                    previewContent.innerHTML = `<iframe src="${fileUrl}" width="100%" height="500px"></iframe>`;
                } else {
                    previewContent.innerHTML = `
                        <div class="text-center">
                            <i class="fas fa-file-alt" style="font-size: 48px; color: #64748b; margin-bottom: 16px;"></i>
                            <p>Bu döküman türü önizlenemiyor</p>
                            <button class="btn btn-primary" onclick="downloadFile()">
                                <i class="fas fa-download"></i>
                                Dosyayı İndir
                            </button>
                        </div>
                    `;
                }
                document.getElementById('imageZoomControls').style.display = 'none';
                break;
            default:
                previewContent.innerHTML = `
                    <div class="text-center">
                        <i class="fas fa-file" style="font-size: 48px; color: #64748b; margin-bottom: 16px;"></i>
                        <p>Bu dosya türü önizlenemiyor</p>
                        <button class="btn btn-primary" onclick="downloadFile()">
                            <i class="fas fa-download"></i>
                            Dosyayı İndir
                        </button>
                    </div>
                `;
                document.getElementById('imageZoomControls').style.display = 'none';
        }
        
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }

    /**
     * Show image preview with zoom functionality
     */
    showImagePreview(imageUrl, imageName) {
        const previewContent = document.getElementById('previewContent');
        const imageZoomControls = document.getElementById('imageZoomControls');
        
        // Show loading state
        previewContent.innerHTML = `
            <div class="image-loading">
                <div class="spinner"></div>
                <div class="loading-text">Resim yükleniyor...</div>
            </div>
        `;
        
        // Show zoom controls
        imageZoomControls.style.display = 'flex';
        
        // Create image element
        const img = new Image();
        img.className = 'preview-image';
        img.alt = imageName;
        
        // Set up zoom functionality
        this.setupImageZoom(img);
        
        // Handle image load
        img.onload = () => {
            previewContent.innerHTML = '';
            previewContent.appendChild(img);
        };
        
        // Handle image error
        img.onerror = () => {
            previewContent.innerHTML = `
                <div class="image-error">
                    <div class="error-icon">
                        <i class="fas fa-exclamation-triangle"></i>
                    </div>
                    <div class="error-text">Resim yüklenemedi</div>
                    <div class="error-description">Dosya bozuk olabilir veya desteklenmeyen bir format olabilir.</div>
                </div>
            `;
        };
        
        // Load the image
        img.src = imageUrl;
    }

    /**
     * Setup image zoom functionality
     */
    setupImageZoom(img) {
        let zoomLevel = 1;
        const minZoom = 0.1;
        const maxZoom = 5;
        const zoomStep = 0.2;
        
        const zoomInBtn = document.getElementById('zoomInBtn');
        const zoomOutBtn = document.getElementById('zoomOutBtn');
        const resetZoomBtn = document.getElementById('resetZoomBtn');
        const zoomLevelDisplay = document.getElementById('zoomLevel');
        
        const updateZoom = () => {
            img.style.transform = `scale(${zoomLevel})`;
            zoomLevelDisplay.textContent = `${Math.round(zoomLevel * 100)}%`;
            
            // Update cursor
            if (zoomLevel > 1) {
                img.classList.add('zoomed');
            } else {
                img.classList.remove('zoomed');
            }
        };
        
        const zoomIn = () => {
            if (zoomLevel < maxZoom) {
                zoomLevel = Math.min(zoomLevel + zoomStep, maxZoom);
                updateZoom();
            }
        };
        
        const zoomOut = () => {
            if (zoomLevel > minZoom) {
                zoomLevel = Math.max(zoomLevel - zoomStep, minZoom);
                updateZoom();
            }
        };
        
        const resetZoom = () => {
            zoomLevel = 1;
            updateZoom();
        };
        
        // Event listeners
        zoomInBtn.onclick = zoomIn;
        zoomOutBtn.onclick = zoomOut;
        resetZoomBtn.onclick = resetZoom;
        
        // Mouse wheel zoom
        img.addEventListener('wheel', (e) => {
            e.preventDefault();
            if (e.deltaY < 0) {
                zoomIn();
            } else {
                zoomOut();
            }
        });
        
        // Click to zoom
        img.addEventListener('click', () => {
            if (zoomLevel === 1) {
                zoomIn();
            } else {
                resetZoom();
            }
        });
        
        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (document.getElementById('filePreviewModal').style.display === 'flex') {
                switch (e.key) {
                    case '+':
                    case '=':
                        e.preventDefault();
                        zoomIn();
                        break;
                    case '-':
                        e.preventDefault();
                        zoomOut();
                        break;
                    case '0':
                        e.preventDefault();
                        resetZoom();
                        break;
                }
            }
        });
        
        // Initialize zoom level
        updateZoom();
    }

    /**
     * Download file
     */
    downloadFile(fileName) {
        const fileUrl = this.filesData.uploadsPath + fileName;
        const link = document.createElement('a');
        link.href = fileUrl;
        link.download = fileName;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    }

    /**
     * Delete file
     */
    deleteFile(fileName) {
        const modal = document.getElementById('deleteModal');
        const deleteFileName = document.getElementById('deleteFileName');
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        
        deleteFileName.textContent = fileName;
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
        
        // Remove existing listeners
        const newConfirmBtn = confirmBtn.cloneNode(true);
        confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);
        
        // Add new listener
        newConfirmBtn.addEventListener('click', () => {
            this.confirmDelete(fileName);
        });
    }

    /**
     * Confirm file deletion
     */
    async confirmDelete(fileName) {
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');
        
        try {
            // Show loading state
            confirmBtn.disabled = true;
            btnText.style.opacity = '0';
            btnLoader.style.display = 'block';
            
            const response = await fetch('/backend/delete-file', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                },
                body: JSON.stringify({ file_name: fileName })
            });
            
            const result = await response.json();
            
            if (result.status === 201) {
                this.closeModal('deleteModal');
                this.showSuccess('Dosya başarıyla silindi.');
                
                // Remove file from local arrays
                this.files = this.files.filter(f => f.name !== fileName);
                this.filteredFiles = this.filteredFiles.filter(f => f.name !== fileName);
                
                // Remove file item from DOM
                const fileItem = document.querySelector(`[data-file-name="${fileName}"]`);
                if (fileItem) {
                    fileItem.remove();
                }
                
                // Check if there are any files left
                const remainingItems = this.filesContainer.querySelectorAll('.file-item');
                if (remainingItems.length === 0) {
                    this.filesContainer.innerHTML = `
                        <div class="empty-state">
                            <div class="empty-icon">
                                <i class="fas fa-folder-open"></i>
                            </div>
                            <h3 class="empty-title">Henüz Dosya Yok</h3>
                            <p class="empty-description">
                                Uploads klasöründe henüz dosya bulunmuyor.<br>
                                Yeni dosya yüklemek için <a href="/panel/dosya-ekle" class="empty-link">dosya yükleme sayfasına</a> gidin.
                            </p>
                        </div>
                    `;
                }
                
                this.updateStats();
                
                // Close preview modal if it's open for this file
                if (this.currentFile && this.currentFile.name === fileName) {
                    this.closeModal('filePreviewModal');
                }
            } else {
                throw new Error(result.message || 'Dosya silinemedi');
            }
        } catch (error) {
            console.error('Delete error:', error);
            this.showError(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
        } finally {
            // Reset button state
            confirmBtn.disabled = false;
            btnText.style.opacity = '1';
            btnLoader.style.display = 'none';
        }
    }

    /**
     * Get file type based on extension
     */
    getFileType(fileName) {
        const ext = fileName.split('.').pop().toLowerCase();
        
        const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'];
        const videoExts = ['mp4', 'avi', 'mov', 'wmv', 'flv', 'webm'];
        const audioExts = ['mp3', 'wav', 'ogg', 'flac'];
        const documentExts = ['doc', 'docx', 'txt', 'rtf'];
        const archiveExts = ['zip', 'rar', '7z', 'tar', 'gz'];
        
        if (imageExts.includes(ext)) return 'image';
        if (videoExts.includes(ext)) return 'video';
        if (audioExts.includes(ext)) return 'audio';
        if (ext === 'pdf') return 'document'; // PDFs are documents, not archives
        if (documentExts.includes(ext)) return 'document';
        if (archiveExts.includes(ext)) return 'archive';
        
        return 'other';
    }

    /**
     * Get file icon based on type
     */
    getFileIcon(fileType, fileName = '') {
        const icons = {
            image: 'fas fa-image',
            video: 'fas fa-video',
            audio: 'fas fa-music',
            document: fileName.toLowerCase().endsWith('.pdf') ? 'fas fa-file-pdf' : 'fas fa-file-word',
            archive: 'fas fa-file-archive',
            other: 'fas fa-file'
        };
        
        return icons[fileType] || icons.other;
    }

    /**
     * Format file size
     */
    formatFileSize(bytes) {
        if (bytes === 0) return '0 B';
        
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        
        // Use appropriate decimal places based on size
        let decimalPlaces = 2;
        if (i === 0) decimalPlaces = 0; // Bytes - no decimals
        if (i === 1) decimalPlaces = 1; // KB - 1 decimal
        if (i >= 2) decimalPlaces = 2;  // MB, GB, TB - 2 decimals
        
        const size = parseFloat((bytes / Math.pow(k, i)).toFixed(decimalPlaces));
        return size + ' ' + sizes[i];
    }

    /**
     * Format date
     */
    formatDate(dateString) {
        const date = new Date(dateString);
        return date.toLocaleDateString('tr-TR', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit'
        });
    }

    /**
     * Update statistics
     */
    updateStats() {
        // Count visible files
        const visibleFiles = this.filesContainer.querySelectorAll('.file-item:not([style*="display: none"])');
        this.filesCount.textContent = `${visibleFiles.length} dosya`;
        
        // Calculate total size of visible files
        let totalSize = 0;
        visibleFiles.forEach(item => {
            const fileName = item.getAttribute('data-file-name');
            const file = this.files.find(f => f.name === fileName);
            if (file) {
                totalSize += file.size;
            }
        });
        
        this.totalSize.textContent = this.formatFileSize(totalSize);
    }


    /**
     * Show success message
     */
    showSuccess(message) {
        this.showAlert(message, 'success');
    }

    /**
     * Show error message
     */
    showError(message) {
        this.showAlert(message, 'error');
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
     * Close modals
     */
    closeModals() {
        const modals = document.querySelectorAll('.modal');
        modals.forEach(modal => {
            modal.classList.remove('show');
            setTimeout(() => {
                modal.style.display = 'none';
            }, 300);
        });
    }

    /**
     * Close specific modal
     */
    closeModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.remove('show');
            setTimeout(() => {
                modal.style.display = 'none';
            }, 300);
        }
    }
}

/**
 * Global functions for template usage
 */
window.closeModal = function(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
    }
};

window.downloadFile = function() {
    if (window.fileManager && window.fileManager.currentFile) {
        window.fileManager.downloadFile(window.fileManager.currentFile.name);
    }
};

window.deleteFile = function() {
    if (window.fileManager && window.fileManager.currentFile) {
        window.fileManager.deleteFile(window.fileManager.currentFile.name);
        window.fileManager.closeModal('filePreviewModal');
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

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.fileManager = new FileManager();
});

// Export for potential external use
window.FileManager = FileManager;
