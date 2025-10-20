/**
 * Sube (Individual Branch View) Page JavaScript
 * Handles delete functionality, image zoom, and interactive elements
 */

class SubeViewManager {
    constructor() {
        this.subeData = window.subeData || {};
        this.currentZoomedImage = null;
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupImageZoom();
        this.setupDeleteButton();
        this.setupSetMainButton();
        this.setupDocumentPreview();
        this.formatDates();
        this.setupCopyToClipboard();
        this.setupKeyboardShortcuts();
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
                const documentModal = document.getElementById('documentPreviewModal');
                
                if (deleteModal && deleteModal.classList.contains('show')) {
                    this.closeDeleteModal();
                } else if (imageModal && imageModal.classList.contains('show')) {
                    this.closeImageModal();
                } else if (documentModal && documentModal.classList.contains('show')) {
                    this.closeDocumentPreviewModal();
                }
            }
        });

        // Close modals when clicking outside
        document.addEventListener('click', (e) => {
            const deleteModal = document.getElementById('deleteModal');
            const imageModal = document.getElementById('imageZoomModal');
            const documentModal = document.getElementById('documentPreviewModal');
            
            if (e.target === deleteModal) {
                this.closeDeleteModal();
            } else if (e.target === imageModal) {
                this.closeImageModal();
            } else if (e.target === documentModal) {
                this.closeDocumentPreviewModal();
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
                const parent = image.closest('.image-preview');
                if (parent) {
                    parent.innerHTML = `
                        <div class="image-error">
                            <i class="icon-image-off"></i>
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

        // Set image source and title
        zoomedImage.src = imageElement.src;
        zoomedImage.alt = imageElement.alt;
        this.currentZoomedImage = imageElement.src;
        
        // Set title based on alt text or default
        const title = imageElement.alt || 'Şube Görseli';
        if (imageTitle) imageTitle.textContent = title;
        
        // Show modal
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
        
        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Close image modal
     */
    closeImageModal() {
        const modal = document.getElementById('imageZoomModal');
        if (!modal) return;

        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
            this.currentZoomedImage = null;
        }, 300);
        
        // Restore body scroll
        document.body.style.overflow = '';
    }

    /**
     * Download current zoomed image
     */
    downloadImage() {
        if (!this.currentZoomedImage) return;

        const link = document.createElement('a');
        link.href = this.currentZoomedImage;
        link.download = this.currentZoomedImage.split('/').pop() || 'sube-gorsel';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        
        this.showAlert('Görsel indiriliyor...', 'success');
    }

    /**
     * Open image in new tab
     */
    openImageInNewTab() {
        if (!this.currentZoomedImage) return;

        window.open(this.currentZoomedImage, '_blank');
    }

    /**
     * Setup delete button functionality
     */
    setupDeleteButton() {
        const deleteBtn = document.getElementById('deleteSubeBtn');
        
        if (!deleteBtn) return;

        deleteBtn.addEventListener('click', () => {
            const id = deleteBtn.getAttribute('data-id');
            const name = deleteBtn.getAttribute('data-name');
            this.showDeleteConfirmation(id, name);
        });
    }

    /**
     * Setup document preview functionality
     */
    setupDocumentPreview() {
        const previewButtons = document.querySelectorAll('.btn-preview-document');
        
        previewButtons.forEach(button => {
            button.addEventListener('click', (e) => {
                e.preventDefault();
                const mid = button.getAttribute('data-mid');
                const mimeType = button.getAttribute('data-mimetype');
                const filePath = button.getAttribute('data-filepath');
                this.openDocumentPreview(mid, mimeType, filePath);
            });
        });
    }

    /**
     * Open document preview modal
     */
    openDocumentPreview(mid, mimeType, filePath) {
        const modal = document.getElementById('documentPreviewModal');
        const container = document.getElementById('documentPreviewContainer');
        const title = document.getElementById('documentPreviewTitle');
        const info = document.getElementById('documentPreviewInfo');
        
        if (!modal || !container) return;

        // Set title and info
        if (title) title.textContent = 'Doküman Önizleme';
        if (info) info.textContent = `Dosya: ${filePath.split('/').pop()}`;

        // Clear previous content
        container.innerHTML = '';

        // Handle different MIME types
        if (mimeType.startsWith('image/')) {
            this.previewImage(container, filePath);
        } else if (mimeType.startsWith('video/')) {
            this.previewVideo(container, filePath, mimeType);
        } else if (mimeType.startsWith('audio/')) {
            this.previewAudio(container, filePath, mimeType);
        } else if (mimeType === 'application/pdf') {
            this.previewPDF(container, filePath);
        } else if (mimeType.startsWith('text/')) {
            this.previewText(container, filePath);
        } else {
            this.previewUnsupported(container, mimeType);
        }

        // Show modal
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
        
        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Preview image files
     */
    previewImage(container, filePath) {
        const img = document.createElement('img');
        img.src = filePath.startsWith('/') ? filePath : '/' + filePath;
        img.alt = 'Doküman Önizleme';
        img.style.maxWidth = '100%';
        img.style.maxHeight = '100%';
        img.style.objectFit = 'contain';
        container.appendChild(img);
    }

    /**
     * Preview video files
     */
    previewVideo(container, filePath, mimeType) {
        const video = document.createElement('video');
        video.src = filePath.startsWith('/') ? filePath : '/' + filePath;
        video.controls = true;
        video.style.maxWidth = '100%';
        video.style.maxHeight = '100%';
        video.style.objectFit = 'contain';
        container.appendChild(video);
    }

    /**
     * Preview audio files
     */
    previewAudio(container, filePath, mimeType) {
        const audio = document.createElement('audio');
        audio.src = filePath.startsWith('/') ? filePath : '/' + filePath;
        audio.controls = true;
        audio.style.width = '100%';
        audio.style.maxWidth = '500px';
        container.appendChild(audio);
    }

    /**
     * Preview PDF files
     */
    previewPDF(container, filePath) {
        const iframe = document.createElement('iframe');
        iframe.src = filePath.startsWith('/') ? filePath : '/' + filePath;
        iframe.style.width = '100%';
        iframe.style.height = '100%';
        iframe.style.border = 'none';
        container.appendChild(iframe);
    }

    /**
     * Preview text files
     */
    previewText(container, filePath) {
        const iframe = document.createElement('iframe');
        iframe.src = filePath.startsWith('/') ? filePath : '/' + filePath;
        iframe.style.width = '100%';
        iframe.style.height = '100%';
        iframe.style.border = 'none';
        container.appendChild(iframe);
    }

    /**
     * Show unsupported file type message
     */
    previewUnsupported(container, mimeType) {
        container.innerHTML = `
            <div class="unsupported-file">
                <i class="fas fa-file"></i>
                <h3>Önizleme Desteklenmiyor</h3>
                <p>Bu dosya türü (${mimeType}) önizlenemiyor. Dosyayı indirmek için "İndir" butonunu kullanın.</p>
            </div>
        `;
    }

    /**
     * Close document preview modal
     */
    closeDocumentPreviewModal() {
        const modal = document.getElementById('documentPreviewModal');
        if (!modal) return;

        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
        
        // Restore body scroll
        document.body.style.overflow = '';
    }

    /**
     * Download current document
     */
    downloadDocument() {
        const modal = document.getElementById('documentPreviewModal');
        const info = document.getElementById('documentPreviewInfo');
        
        if (!modal || !info) return;

        // Extract file path from info text
        const infoText = info.textContent;
        const filePath = infoText.split('Dosya: ')[1];
        
        if (filePath) {
            const link = document.createElement('a');
            link.href = filePath.startsWith('/') ? filePath : '/' + filePath;
            link.download = filePath.split('/').pop() || 'document';
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            
            this.showAlert('Dosya indiriliyor...', 'success');
        }
    }

    /**
     * Open document in new tab
     */
    openDocumentInNewTab() {
        const modal = document.getElementById('documentPreviewModal');
        const info = document.getElementById('documentPreviewInfo');
        
        if (!modal || !info) return;

        // Extract file path from info text
        const infoText = info.textContent;
        const filePath = infoText.split('Dosya: ')[1];
        
        if (filePath) {
            window.open(filePath.startsWith('/') ? filePath : '/' + filePath, '_blank');
        }
    }

    /**
     * Setup set as main branch button
     */
    setupSetMainButton() {
        const setMainBtn = document.getElementById('setMainSubeBtn');
        if (!setMainBtn) return;

        setMainBtn.addEventListener('click', async (e) => {
            e.preventDefault();
            e.stopPropagation();
            const sid = setMainBtn.getAttribute('data-id');
            const btnText = setMainBtn.querySelector('.btn-text');
            const btnLoader = setMainBtn.querySelector('.btn-loader');

            try {
                // Show loading state
                setMainBtn.disabled = true;
                if (btnText) btnText.style.opacity = '0';
                if (btnLoader) btnLoader.style.display = 'block';
                const url = `/backend/sube/${sid}/set-as-main`;
                console.debug('Setting sube as main:', url);
                const response = await fetch(url, {
                    method: 'POST',
                    headers: {
                        'X-Requested-With': 'XMLHttpRequest'
                    },
                    body: JSON.stringify({})
                });

                const result = await response.json().catch(() => ({}));

                if (result.status === 201) {
                    this.showAlert('Şube ana şube olarak ayarlandı.', 'success');
                    // Reload to reflect new state
                    setTimeout(() => {
                        window.location.reload();
                    }, 1200);
                } else {
                    this.showAlert(result.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
                }
            } catch (error) {
                console.error('Set as main error:', error);
                this.showAlert('Bağlantı hatası oluştu.', 'error');
            } finally {
                setMainBtn.disabled = false;
                if (btnText) btnText.style.opacity = '1';
                if (btnLoader) btnLoader.style.display = 'none';
            }
        });
    }

    /**
     * Show delete confirmation modal
     */
    showDeleteConfirmation(id, name) {
        const modal = document.getElementById('deleteModal');
        const nameSpan = document.getElementById('deleteItemName');
        const confirmBtn = document.getElementById('confirmDeleteBtn');

        if (!modal || !nameSpan || !confirmBtn) return;

        nameSpan.textContent = name;
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);

        // Remove existing listeners
        const newConfirmBtn = confirmBtn.cloneNode(true);
        confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);

        // Add new listener
        newConfirmBtn.addEventListener('click', () => {
            this.deleteSube(id);
        });

        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Delete branch
     */
    async deleteSube(id) {
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');

        if (!confirmBtn || !btnText || !btnLoader) return;

        try {
            // Show loading state
            confirmBtn.disabled = true;
            btnText.style.opacity = '0';
            btnLoader.style.display = 'block';

            const response = await fetch(`/backend/sube/${id}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const responseData = await response.json().catch(() => ({}));
            
            if (responseData.status === 201) {
                this.closeDeleteModal();
                this.showAlert('Şube başarıyla silindi.', 'success');
                
                // Redirect to branches list after 2 seconds
                setTimeout(() => {
                    window.location.href = '/panel/subeler';
                }, 2000);

            } else {
                this.showAlert(responseData.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            }
        } catch (error) {
            console.error('Delete error:', error);
            this.showAlert('Bağlantı hatası oluştu.', 'error');
        } finally {
            // Reset button state
            confirmBtn.disabled = false;
            btnText.style.opacity = '1';
            btnLoader.style.display = 'none';
        }
    }

    /**
     * Close delete modal
     */
    closeDeleteModal() {
        const modal = document.getElementById('deleteModal');
        if (!modal) return;

        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
        
        // Restore body scroll
        document.body.style.overflow = '';
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
                    const formattedDate = date.toLocaleDateString('tr-TR', {
                        year: 'numeric',
                        month: '2-digit',
                        day: '2-digit',
                        hour: '2-digit',
                        minute: '2-digit'
                    });
                    element.textContent = formattedDate;
                } catch (error) {
                    console.warn('Date formatting error:', error);
                }
            }
        });
    }

    /**
     * Setup copy to clipboard functionality
     */
    setupCopyToClipboard() {
        // Add copy buttons to contact links
        const contactLinks = document.querySelectorAll('.contact-link');
        contactLinks.forEach(link => {
            const copyBtn = document.createElement('button');
            copyBtn.className = 'btn-icon copy-btn';
            copyBtn.innerHTML = '<i class="icon-copy"></i>';
            copyBtn.title = 'Kopyala';
            copyBtn.style.marginLeft = '8px';
            
            copyBtn.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                const text = link.textContent.trim();
                this.copyToClipboard(text, 'İletişim bilgisi kopyalandı!');
            });
            
            link.parentNode.appendChild(copyBtn);
        });

        // Add copy functionality to address
        const addressElements = document.querySelectorAll('.info-value');
        addressElements.forEach(element => {
            const label = element.parentNode.querySelector('.info-label');
            if (label && label.textContent.includes('Adres')) {
                element.style.cursor = 'pointer';
                element.title = 'Kopyalamak için tıklayın';
                
                element.addEventListener('click', () => {
                    this.copyToClipboard(element.textContent, 'Adres kopyalandı!');
                });
            }
        });
    }

    /**
     * Copy text to clipboard
     */
    async copyToClipboard(text, successMessage = 'Kopyalandı!') {
        try {
            if (navigator.clipboard && window.isSecureContext) {
                await navigator.clipboard.writeText(text);
            } else {
                // Fallback for older browsers
                const textArea = document.createElement('textarea');
                textArea.value = text;
                textArea.style.position = 'fixed';
                textArea.style.left = '-999999px';
                textArea.style.top = '-999999px';
                document.body.appendChild(textArea);
                textArea.focus();
                textArea.select();
                document.execCommand('copy');
                textArea.remove();
            }
            
            this.showAlert(successMessage, 'success');
        } catch (error) {
            console.error('Copy to clipboard failed:', error);
            this.showAlert('Kopyalama işlemi başarısız oldu.', 'error');
        }
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
     * Handle window resize
     */
    handleResize() {
        // Close any open modals on mobile
        if (window.innerWidth < 768) {
            const imageModal = document.getElementById('imageZoomModal');
            if (imageModal?.classList.contains('show')) {
                this.closeImageModal();
            }
        }
    }

    /**
     * Setup keyboard shortcuts
     */
    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            // Ctrl/Cmd + E for edit
            if ((e.ctrlKey || e.metaKey) && e.key === 'e') {
                e.preventDefault();
                const editBtn = document.querySelector('a[href*="/duzenle"]');
                if (editBtn) {
                    editBtn.click();
                }
            }
            
            // Ctrl/Cmd + D for delete
            if ((e.ctrlKey || e.metaKey) && e.key === 'd') {
                e.preventDefault();
                const deleteBtn = document.getElementById('deleteSubeBtn');
                if (deleteBtn) {
                    deleteBtn.click();
                }
            }
            
            // Ctrl/Cmd + B for back to list
            if ((e.ctrlKey || e.metaKey) && e.key === 'b') {
                e.preventDefault();
                window.location.href = '/panel/subeler';
            }
        });
    }

    /**
     * Setup tooltips for truncated text
     */
    setupTooltips() {
        const truncatedElements = document.querySelectorAll('.truncate');
        
        truncatedElements.forEach(element => {
            if (element.scrollWidth > element.clientWidth) {
                element.title = element.textContent;
                element.style.cursor = 'help';
            }
        });
    }

    /**
     * Format bytes for display
     */
    formatBytes(bytes) {
        if (bytes === 0) return '0 Bytes';
        
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    /**
     * Animate elements on scroll
     */
    setupScrollAnimations() {
        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    entry.target.classList.add('animate-in');
                }
            });
        }, {
            threshold: 0.1,
            rootMargin: '0px 0px -50px 0px'
        });

        const cards = document.querySelectorAll('.content-card');
        cards.forEach(card => {
            observer.observe(card);
        });
    }
}

/**
 * Global functions for template usage
 */
window.closeDeleteModal = function() {
    const modal = document.getElementById('deleteModal');
    if (modal) {
        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
        document.body.style.overflow = '';
    }
};

window.closeImageModal = function() {
    const modal = document.getElementById('imageZoomModal');
    if (modal) {
        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
        document.body.style.overflow = '';
    }
};

window.downloadImage = function() {
    const manager = window.subeViewManager;
    if (manager) {
        manager.downloadImage();
    }
};

window.openImageInNewTab = function() {
    const manager = window.subeViewManager;
    if (manager) {
        manager.openImageInNewTab();
    }
};

window.closeDocumentPreviewModal = function() {
    const manager = window.subeViewManager;
    if (manager) {
        manager.closeDocumentPreviewModal();
    }
};

window.downloadDocument = function() {
    const manager = window.subeViewManager;
    if (manager) {
        manager.downloadDocument();
    }
};

window.openDocumentInNewTab = function() {
    const manager = window.subeViewManager;
    if (manager) {
        manager.openDocumentInNewTab();
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

/**
 * Utility functions
 */
function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('tr-TR', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

function formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.subeViewManager = new SubeViewManager();
    
    // Add CSS for animations
    const style = document.createElement('style');
    style.textContent = `
        .content-card {
            opacity: 1;
            transform: translateY(0);
            transition: all 0.6s ease;
        }
        
        .content-card.animate-in {
            opacity: 1;
            transform: translateY(0);
        }
        
        .image-error {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            padding: 20px;
            background: var(--bg-tertiary);
            border: 1px dashed var(--border-color);
            border-radius: var(--border-radius);
            color: var(--text-muted);
            font-size: 14px;
            gap: 8px;
        }
        
        .image-error i {
            font-size: 24px;
        }
        
        .copy-btn {
            opacity: 0;
            transition: opacity 0.2s ease;
            border: 1px solid var(--border-color);
            background: var(--bg-tertiary);
            border-radius: var(--border-radius);
            padding: 4px 8px;
            cursor: pointer;
            color: var(--text-secondary);
        }
        
        .copy-btn:hover {
            background: var(--primary-color);
            color: var(--text-white);
            border-color: var(--primary-color);
        }
        
        .info-item:hover .copy-btn {
            opacity: 1;
        }
        
        .loading-overlay {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(255, 255, 255, 0.8);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 9999;
        }
        
        .loading-spinner {
            width: 40px;
            height: 40px;
            border: 4px solid var(--border-color);
            border-top: 4px solid var(--primary-color);
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
    `;
    document.head.appendChild(style);
});

// Export for potential external use
window.SubeViewManager = SubeViewManager;
