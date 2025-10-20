/**
 * Frontend Sube (Branch) Page JavaScript
 * Handles working hours formatting, document preview, and interactive elements
 */

function decodeHtmlEntities(encoded) {
    const txt = document.createElement('textarea');
    txt.innerHTML = encoded;
    return txt.value;
}

function formatWorkingHours(workingHoursJson, element) {
    if (workingHoursJson && typeof workingHoursJson === 'object') {
        let formattedHours = '';
        
        // Format the JSON data into readable text
        for (const [day, hours] of Object.entries(workingHoursJson)) {
            if (hours && hours.trim() !== '') {
                formattedHours += day + ': ' + hours + '<br>';
            }
        }
        
        if (formattedHours) {
            element.innerHTML = formattedHours;
        }
    }
}

// Format file size function
function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

// Document preview function
function previewDocument(filePath, mimeType, fileName) {
    // Create modal overlay
    const modal = document.createElement('div');
    modal.id = 'documentPreviewModal';
    modal.style.cssText = `
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.8);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 9999;
        opacity: 0;
        transition: opacity 0.3s ease;
    `;
    
    // Create modal content
    const modalContent = document.createElement('div');
    modalContent.style.cssText = `
        background: white;
        border-radius: 12px;
        max-width: 90vw;
        max-height: 90vh;
        width: 90vw;
        height: 90vh;
        display: flex;
        flex-direction: column;
        overflow: hidden;
        box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
    `;
    
    // Create header
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 16px 20px;
        background: #f8fafc;
        border-bottom: 1px solid #e2e8f0;
        display: flex;
        justify-content: space-between;
        align-items: center;
        flex-shrink: 0;
    `;
    
    const title = document.createElement('h3');
    title.textContent = fileName;
    title.style.cssText = `
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: #1f2937;
    `;
    
    const closeBtn = document.createElement('button');
    closeBtn.innerHTML = '<i class="fas fa-times"></i>';
    closeBtn.style.cssText = `
        background: #ef4444;
        color: white;
        border: none;
        border-radius: 6px;
        width: 32px;
        height: 32px;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: background 0.2s;
    `;
    closeBtn.onclick = () => closeDocumentModal();
    
    header.appendChild(title);
    header.appendChild(closeBtn);
    
    // Create body
    const body = document.createElement('div');
    body.style.cssText = `
        flex: 1;
        padding: 0;
        overflow: hidden;
        display: flex;
        align-items: center;
        justify-content: center;
        background: #f8fafc;
    `;
    
    // Create preview content based on MIME type
    let previewElement;
    
    if (mimeType.startsWith('image/')) {
        previewElement = document.createElement('img');
        previewElement.src = filePath.startsWith('/') ? filePath : '/' + filePath;
        previewElement.style.cssText = 'max-width: 100%; max-height: 100%; object-fit: contain;';
    } else if (mimeType.startsWith('video/')) {
        previewElement = document.createElement('video');
        previewElement.src = filePath.startsWith('/') ? filePath : '/' + filePath;
        previewElement.controls = true;
        previewElement.style.cssText = 'max-width: 100%; max-height: 100%; object-fit: contain;';
    } else if (mimeType.startsWith('audio/')) {
        previewElement = document.createElement('audio');
        previewElement.src = filePath.startsWith('/') ? filePath : '/' + filePath;
        previewElement.controls = true;
        previewElement.style.cssText = 'width: 100%; max-width: 500px;';
    } else if (mimeType === 'application/pdf') {
        previewElement = document.createElement('iframe');
        previewElement.src = filePath.startsWith('/') ? filePath : '/' + filePath;
        previewElement.style.cssText = 'width: 100%; height: 100%; border: none;';
    } else if (mimeType.startsWith('text/')) {
        previewElement = document.createElement('iframe');
        previewElement.src = filePath.startsWith('/') ? filePath : '/' + filePath;
        previewElement.style.cssText = 'width: 100%; height: 100%; border: none;';
    } else {
        previewElement = document.createElement('div');
        previewElement.style.cssText = `
            text-align: center;
            padding: 40px;
            color: #6b7280;
        `;
        previewElement.innerHTML = `
            <i class="fas fa-file" style="font-size: 48px; margin-bottom: 16px; color: #9ca3af;"></i>
            <h3 style="margin: 0 0 8px 0; color: #374151;">Önizleme Desteklenmiyor</h3>
            <p style="margin: 0; color: #6b7280;">Bu dosya türü (${mimeType}) önizlenemiyor.</p>
        `;
    }
    
    body.appendChild(previewElement);
    
    // Assemble modal
    modalContent.appendChild(header);
    modalContent.appendChild(body);
    modal.appendChild(modalContent);
    
    // Add to page
    document.body.appendChild(modal);
    document.body.style.overflow = 'hidden';
    
    // Show modal
    setTimeout(() => {
        modal.style.opacity = '1';
    }, 10);
    
    // Close on escape key
    const handleEscape = (e) => {
        if (e.key === 'Escape') {
            closeDocumentModal();
        }
    };
    document.addEventListener('keydown', handleEscape);
    
    // Store cleanup function
    modal._cleanup = () => {
        document.removeEventListener('keydown', handleEscape);
    };
}

// Close document modal
function closeDocumentModal() {
    const modal = document.getElementById('documentPreviewModal');
    if (modal) {
        if (modal._cleanup) {
            modal._cleanup();
        }
        modal.style.opacity = '0';
        setTimeout(() => {
            document.body.removeChild(modal);
            document.body.style.overflow = '';
        }, 300);
    }
}

// Download document function
function downloadDocument(filePath, fileName) {
    const link = document.createElement('a');
    link.href = filePath.startsWith('/') ? filePath : '/' + filePath;
    link.download = fileName;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    // Parse and format working hours JSON
    const workingHoursElements = document.querySelectorAll('.working-hours-content, .working-hours-sidebar');
    
    workingHoursElements.forEach(function(element) {
        try {
            // Get data from global variable set by template
            const escaped = window.workingHoursData || '';
            
            const decoded = decodeHtmlEntities(escaped);
            
            // Check if the decoded string is valid JSON
            if (decoded && decoded.trim() !== '' && decoded !== 'null' && decoded !== 'undefined') {
                // Try to clean the string first
                let cleanJson = decoded.trim();
                
                // Remove any HTML entities or extra characters
                cleanJson = cleanJson.replace(/&quot;/g, '"').replace(/&amp;/g, '&').replace(/&lt;/g, '<').replace(/&gt;/g, '>');
                
                // Check if it looks like JSON (either direct JSON or JSON wrapped in quotes)
                if (cleanJson.startsWith('{') && cleanJson.endsWith('}')) {
                    // Direct JSON
                    const workingHoursJson = JSON.parse(cleanJson);
                    formatWorkingHours(workingHoursJson, element);
                } else if (cleanJson.startsWith('"{') && cleanJson.endsWith('}"')) {
                    // JSON wrapped in quotes - remove the outer quotes
                    const unwrappedJson = cleanJson.slice(1, -1);
                    const workingHoursJson = JSON.parse(unwrappedJson);
                    formatWorkingHours(workingHoursJson, element);
                } else {
                    // If it's not JSON format, just display the raw text
                    element.innerHTML = decoded;
                }
            } else {
                // Fallback: try to get from data attribute or original content
                const rawHours = element.getAttribute('data-raw-hours') || element.textContent;
                if (rawHours && rawHours.trim() !== '') {
                    element.innerHTML = rawHours;
                }
            }
        } catch (e) {
            // If JSON parsing fails, keep the original content
            const rawHours = element.getAttribute('data-raw-hours') || element.textContent;
            if (rawHours && rawHours.trim() !== '') {
                element.innerHTML = rawHours;
            }
        }
    });
    
    // Add hover effects for document items
    const documentItems = document.querySelectorAll('.document-item');
    documentItems.forEach(item => {
        item.addEventListener('mouseenter', function() {
            this.style.transform = 'translateY(-2px)';
            this.style.boxShadow = '0 4px 12px rgba(0, 0, 0, 0.15)';
        });
        
        item.addEventListener('mouseleave', function() {
            this.style.transform = 'translateY(0)';
            this.style.boxShadow = 'none';
        });
    });
    
    // Format file sizes
    const fileSizeElements = document.querySelectorAll('.file-size');
    fileSizeElements.forEach(element => {
        const sizeInBytes = parseInt(element.getAttribute('data-size'));
        if (!isNaN(sizeInBytes)) {
            element.textContent = formatFileSize(sizeInBytes);
        }
    });
});
