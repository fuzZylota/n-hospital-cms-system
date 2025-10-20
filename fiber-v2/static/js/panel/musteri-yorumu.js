/**
 * Musteri Yorumu (Individual Testimonial View) Page JavaScript
 * Handles delete functionality, image zoom, and interactive elements
 */

class TestimonialViewManager {
    constructor() {
        this.testimonialData = window.testimonialData || {};
        this.currentZoomedImage = null;
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupImageZoom();
        this.setupDeleteButton();
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
                
                if (deleteModal.classList.contains('show')) {
                    this.closeDeleteModal();
                } else if (imageModal.classList.contains('show')) {
                    this.closeImageModal();
                }
            }
        });

        // Close modals when clicking outside
        document.addEventListener('click', (e) => {
            const deleteModal = document.getElementById('deleteModal');
            const imageModal = document.getElementById('imageZoomModal');
            
            if (e.target === deleteModal) {
                this.closeDeleteModal();
            } else if (e.target === imageModal) {
                this.closeImageModal();
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
        const title = imageElement.alt || 'Müşteri Fotoğrafı';
        imageTitle.textContent = title;
        
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
        link.download = this.currentZoomedImage.split('/').pop() || 'customer-image';
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
        const deleteBtn = document.getElementById('deleteTestimonialBtn');
        
        if (!deleteBtn) return;

        deleteBtn.addEventListener('click', () => {
            const id = deleteBtn.getAttribute('data-id');
            const name = deleteBtn.getAttribute('data-name');
            this.showDeleteConfirmation(id, name);
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
            this.deleteTestimonial(id);
        });

        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Delete testimonial
     */
    async deleteTestimonial(id) {
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');

        if (!confirmBtn || !btnText || !btnLoader) return;

        try {
            // Show loading state
            confirmBtn.disabled = true;
            btnText.style.opacity = '0';
            btnLoader.style.display = 'block';

            const response = await fetch(`/backend/testimonial/${id}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const responseData = await response.json().catch(() => ({}));
            
            if (responseData.status === 201) {
                this.closeDeleteModal();
                this.showAlert('Müşteri yorumu başarıyla silindi.', 'success');
                
                // Redirect to testimonials list after 2 seconds
                setTimeout(() => {
                    window.location.href = '/panel/musteri-yorumlari';
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
        // Add copy button to testimonial ID
        const idBadge = document.querySelector('.id-badge');
        if (idBadge) {
            idBadge.style.cursor = 'pointer';
            idBadge.title = 'Kopyalamak için tıklayın';
            
            idBadge.addEventListener('click', () => {
                this.copyToClipboard(idBadge.textContent, 'Testimonial ID kopyalandı!');
            });
        }

        // Add copy button to testimonial content
        const contentText = document.querySelector('.content-text');
        if (contentText) {
            const copyBtn = document.createElement('button');
            copyBtn.className = 'btn-icon copy-btn';
            copyBtn.innerHTML = '<i class="icon-copy"></i>';
            copyBtn.title = 'Yorumu Kopyala';
            copyBtn.style.position = 'absolute';
            copyBtn.style.top = '10px';
            copyBtn.style.right = '10px';
            copyBtn.style.opacity = '0';
            copyBtn.style.transition = 'opacity 0.2s ease';
            
            copyBtn.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                this.copyToClipboard(contentText.textContent.trim(), 'Yorum kopyalandı!');
            });
            
            contentText.style.position = 'relative';
            contentText.appendChild(copyBtn);
            
            // Show copy button on hover
            contentText.addEventListener('mouseenter', () => {
                copyBtn.style.opacity = '1';
            });
            
            contentText.addEventListener('mouseleave', () => {
                copyBtn.style.opacity = '0';
            });
        }
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
                const deleteBtn = document.getElementById('deleteTestimonialBtn');
                if (deleteBtn) {
                    deleteBtn.click();
                }
            }
            
            // Ctrl/Cmd + B for back to list
            if ((e.ctrlKey || e.metaKey) && e.key === 'b') {
                e.preventDefault();
                window.location.href = '/panel/musteri-yorumlari';
            }

            // Ctrl/Cmd + C for copy content
            if ((e.ctrlKey || e.metaKey) && e.key === 'c' && !window.getSelection().toString()) {
                e.preventDefault();
                const contentText = document.querySelector('.content-text');
                if (contentText) {
                    this.copyToClipboard(contentText.textContent.trim(), 'Yorum kopyalandı!');
                }
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
}

/**
 * Global functions for template usage
 */
window.closeDeleteModal = function() {
    const modal = document.getElementById('deleteModal');
    modal.classList.remove('show');
    setTimeout(() => {
        modal.style.display = 'none';
    }, 300);
    document.body.style.overflow = '';
};

window.closeImageModal = function() {
    const modal = document.getElementById('imageZoomModal');
    modal.classList.remove('show');
    setTimeout(() => {
        modal.style.display = 'none';
    }, 300);
    document.body.style.overflow = '';
};

window.downloadImage = function() {
    const manager = window.testimonialViewManager;
    if (manager) {
        manager.downloadImage();
    }
};

window.openImageInNewTab = function() {
    const manager = window.testimonialViewManager;
    if (manager) {
        manager.openImageInNewTab();
    }
};

window.closeAlert = function(alertId) {
    const alert = document.getElementById(alertId);
    alert.classList.remove('show');
    setTimeout(() => {
        alert.style.display = 'none';
    }, 300);
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
    window.testimonialViewManager = new TestimonialViewManager();
    
    // Add CSS for animations and interactions
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
            background: var(--bg-primary);
            border: 1px solid var(--border-color);
            border-radius: var(--border-radius);
            width: 32px;
            height: 32px;
            display: flex;
            align-items: center;
            justify-content: center;
            cursor: pointer;
            color: var(--text-secondary);
        }
        
        .copy-btn:hover {
            background: var(--primary-color);
            color: var(--text-white);
            border-color: var(--primary-color);
        }
        
        .info-item:hover .copy-btn,
        .content-text:hover .copy-btn {
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
        
        .btn-icon {
            padding: 4px 8px;
            border: 1px solid var(--border-color);
            background: var(--bg-tertiary);
            border-radius: var(--border-radius);
            cursor: pointer;
            transition: var(--transition);
            color: var(--text-secondary);
        }
        
        .btn-icon:hover {
            background: var(--primary-color);
            color: var(--text-white);
            border-color: var(--primary-color);
        }
    `;
    document.head.appendChild(style);
});

// Export for potential external use
window.TestimonialViewManager = TestimonialViewManager;
