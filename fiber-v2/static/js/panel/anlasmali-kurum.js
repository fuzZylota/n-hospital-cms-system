/**
 * Anlasmali Kurum (Individual Contract Partner View) Page JavaScript
 * Handles image zoom, delete confirmation, copy to clipboard, and keyboard shortcuts
 */

class AnlasmaliKurumViewManager {
    constructor() {
        this.currentImage = null;
        this.currentImageSrc = null;
        this.currentImageTitle = null;
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupImageZoom();
        this.setupDeleteButton();
        this.formatDates();
        this.setupCopyToClipboard();
        this.handleResize();
        this.setupKeyboardShortcuts();
        this.setupTooltips();
        this.setupScrollAnimations();
    }

    setupEventListeners() {
        // Image zoom functionality
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('zoomable') || e.target.closest('.image-preview')) {
                const imageElement = e.target.classList.contains('zoomable') ? e.target : e.target.closest('.image-preview').querySelector('.zoomable');
                if (imageElement) {
                    this.openImageModal(imageElement);
                }
            }
        });

        // Modal close functionality
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) {
                this.closeImageModal();
            }
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.closeImageModal();
            }
        });

        // Window resize
        window.addEventListener('resize', () => {
            this.handleResize();
        });
    }

    setupImageZoom() {
        // Add zoom cursor to zoomable images
        const zoomableImages = document.querySelectorAll('.zoomable');
        zoomableImages.forEach(img => {
            img.style.cursor = 'zoom-in';
        });
    }

    openImageModal(imageElement) {
        this.currentImage = imageElement;
        this.currentImageSrc = imageElement.src;
        this.currentImageTitle = imageElement.title || imageElement.alt || 'Görsel';

        const modal = document.getElementById('imageZoomModal');
        const zoomedImage = document.getElementById('zoomedImage');
        const imageTitle = document.getElementById('imageTitle');

        if (modal && zoomedImage && imageTitle) {
            zoomedImage.src = this.currentImageSrc;
            zoomedImage.alt = imageElement.alt || 'Zoomed Image';
            imageTitle.textContent = this.currentImageTitle;

            modal.style.display = 'flex';
            setTimeout(() => {
                modal.classList.add('show');
            }, 10);
        }
    }

    closeImageModal() {
        const modal = document.getElementById('imageZoomModal');
        if (modal) {
            modal.classList.remove('show');
            setTimeout(() => {
                modal.style.display = 'none';
                this.currentImage = null;
                this.currentImageSrc = null;
                this.currentImageTitle = null;
            }, 300);
        }
    }

    downloadImage() {
        if (this.currentImageSrc) {
            const link = document.createElement('a');
            link.href = this.currentImageSrc;
            link.download = this.currentImageTitle || 'image';
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
        }
    }

    openImageInNewTab() {
        if (this.currentImageSrc) {
            window.open(this.currentImageSrc, '_blank');
        }
    }

    setupDeleteButton() {
        const deleteBtn = document.getElementById('deleteAnlasmaliKurumBtn');
        if (deleteBtn) {
            deleteBtn.addEventListener('click', (e) => {
                e.preventDefault();
                const id = deleteBtn.getAttribute('data-id');
                const name = deleteBtn.getAttribute('data-name');
                this.showDeleteConfirmation(id, name);
            });
        }
    }

    showDeleteConfirmation(id, name) {
        const modal = document.getElementById('deleteModal');
        const deleteItemName = document.getElementById('deleteItemName');
        const confirmBtn = document.getElementById('confirmDeleteBtn');

        if (modal && deleteItemName && confirmBtn) {
            deleteItemName.textContent = name;
            modal.style.display = 'flex';
            setTimeout(() => {
                modal.classList.add('show');
            }, 10);

            // Remove existing event listeners
            const newConfirmBtn = confirmBtn.cloneNode(true);
            confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);

            // Add new event listener
            newConfirmBtn.addEventListener('click', () => {
                this.deleteAnlasmaliKurum(id);
            });
        }
    }

    async deleteAnlasmaliKurum(id) {
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');

        if (confirmBtn && btnText && btnLoader) {
            // Show loading state
            confirmBtn.disabled = true;
            btnText.style.opacity = '0';
            btnLoader.style.display = 'block';

            try {
                const response = await fetch(`/backend/anlasmali-kurum/${id}/delete`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                });

                const result = await response.json();

                if (result.status === 201) {
                    this.showAlert('Anlaşmalı kurum başarıyla silindi.', 'success');
                    setTimeout(() => {
                        window.location.href = '/panel/anlasmali-kurumlar';
                    }, 1500);
                } else {
                    this.showAlert(result.message || 'Bir hata oluştu.', 'error');
                }
            } catch (error) {
                console.error('Delete error:', error);
                this.showAlert('Bir hata oluştu. Lütfen tekrar deneyin.', 'error');
            } finally {
                // Hide loading state
                confirmBtn.disabled = false;
                btnText.style.opacity = '1';
                btnLoader.style.display = 'none';
                this.closeDeleteModal();
            }
        }
    }

    closeDeleteModal() {
        const modal = document.getElementById('deleteModal');
        if (modal) {
            modal.classList.remove('show');
            setTimeout(() => {
                modal.style.display = 'none';
            }, 300);
        }
    }

    formatDates() {
        // Format any date elements if needed
        const dateElements = document.querySelectorAll('[data-date]');
        dateElements.forEach(element => {
            const dateString = element.getAttribute('data-date');
            if (dateString) {
                const formattedDate = this.formatDate(dateString);
                element.textContent = formattedDate;
            }
        });
    }

    setupCopyToClipboard() {
        // Add copy functionality to contact links
        const contactLinks = document.querySelectorAll('.contact-link');
        contactLinks.forEach(link => {
            link.addEventListener('click', (e) => {
                // Don't prevent default for mailto/tel links
                if (link.href.startsWith('mailto:') || link.href.startsWith('tel:')) {
                    return;
                }

                e.preventDefault();
                const text = link.textContent.trim();
                this.copyToClipboard(text, 'Kopyalandı!');
            });
        });

        // Add copy functionality to info values
        const infoValues = document.querySelectorAll('.info-value');
        infoValues.forEach(value => {
            value.style.cursor = 'pointer';
            value.title = 'Kopyalamak için tıklayın';
            
            value.addEventListener('click', () => {
                const text = value.textContent.trim();
                if (text) {
                    this.copyToClipboard(text, 'Kopyalandı!');
                }
            });
        });
    }

    async copyToClipboard(text, successMessage = 'Kopyalandı!') {
        try {
            await navigator.clipboard.writeText(text);
            this.showAlert(successMessage, 'success');
        } catch (err) {
            console.error('Copy failed:', err);
            // Fallback for older browsers
            const textArea = document.createElement('textarea');
            textArea.value = text;
            document.body.appendChild(textArea);
            textArea.select();
            try {
                document.execCommand('copy');
                this.showAlert(successMessage, 'success');
            } catch (fallbackErr) {
                console.error('Fallback copy failed:', fallbackErr);
                this.showAlert('Kopyalama başarısız.', 'error');
            }
            document.body.removeChild(textArea);
        }
    }

    showAlert(message, type = 'success') {
        const alertId = type === 'success' ? 'successMessage' : 'errorMessage';
        const alert = document.getElementById(alertId);
        const alertText = alert.querySelector('.alert-text');

        if (alert && alertText) {
            alertText.textContent = message;
            alert.style.display = 'flex';
            setTimeout(() => {
                alert.classList.add('show');
            }, 10);

            // Auto-hide after 3 seconds
            setTimeout(() => {
                this.closeAlert(alertId);
            }, 3000);
        }
    }

    closeAlert(alertId) {
        const alert = document.getElementById(alertId);
        if (alert) {
            alert.classList.remove('show');
            setTimeout(() => {
                alert.style.display = 'none';
            }, 300);
        }
    }

    handleResize() {
        // Handle responsive adjustments
        const isMobile = window.innerWidth <= 768;
        const actionButtons = document.querySelector('.action-buttons');
        
        if (actionButtons) {
            if (isMobile) {
                actionButtons.style.flexDirection = 'column';
                actionButtons.style.width = '100%';
            } else {
                actionButtons.style.flexDirection = 'row';
                actionButtons.style.width = 'auto';
            }
        }
    }

    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            // Ctrl/Cmd + K to focus search (if exists)
            if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                e.preventDefault();
                const searchInput = document.querySelector('input[type="search"]');
                if (searchInput) {
                    searchInput.focus();
                }
            }

            // Escape to close modals
            if (e.key === 'Escape') {
                this.closeImageModal();
                this.closeDeleteModal();
            }

            // Ctrl/Cmd + Enter to submit forms
            if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
                const submitBtn = document.querySelector('button[type="submit"]');
                if (submitBtn && !submitBtn.disabled) {
                    submitBtn.click();
                }
            }
        });
    }

    setupTooltips() {
        // Add tooltips to interactive elements
        const tooltipElements = document.querySelectorAll('[title]');
        tooltipElements.forEach(element => {
            element.addEventListener('mouseenter', (e) => {
                // Custom tooltip implementation if needed
            });
        });
    }

    formatBytes(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    setupScrollAnimations() {
        // Intersection Observer for scroll animations
        const observerOptions = {
            threshold: 0.1,
            rootMargin: '0px 0px -50px 0px'
        };

        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    entry.target.style.opacity = '1';
                    entry.target.style.transform = 'translateY(0)';
                }
            });
        }, observerOptions);

        // Observe content cards
        const contentCards = document.querySelectorAll('.content-card');
        contentCards.forEach(card => {
            card.style.opacity = '0';
            card.style.transform = 'translateY(20px)';
            card.style.transition = 'opacity 0.6s ease, transform 0.6s ease';
            observer.observe(card);
        });
    }
}

// Global functions for modal management
function closeImageModal() {
    if (window.anlasmaliKurumManager) {
        window.anlasmaliKurumManager.closeImageModal();
    }
}

function closeDeleteModal() {
    if (window.anlasmaliKurumManager) {
        window.anlasmaliKurumManager.closeDeleteModal();
    }
}

function downloadImage() {
    if (window.anlasmaliKurumManager) {
        window.anlasmaliKurumManager.downloadImage();
    }
}

function openImageInNewTab() {
    if (window.anlasmaliKurumManager) {
        window.anlasmaliKurumManager.openImageInNewTab();
    }
}

function closeAlert(alertId) {
    if (window.anlasmaliKurumManager) {
        window.anlasmaliKurumManager.closeAlert(alertId);
    }
}

// Utility functions
function formatDate(dateString) {
    try {
        const date = new Date(dateString);
        return date.toLocaleDateString('tr-TR', {
            year: 'numeric',
            month: 'long',
            day: 'numeric'
        });
    } catch (error) {
        return dateString;
    }
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
    window.anlasmaliKurumManager = new AnlasmaliKurumViewManager();
});

// Export for potential external use
window.AnlasmaliKurumViewManager = AnlasmaliKurumViewManager;
