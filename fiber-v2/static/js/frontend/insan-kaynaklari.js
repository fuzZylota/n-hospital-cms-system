// İnsan Kaynakları Page JavaScript

document.addEventListener('DOMContentLoaded', function() {
    const form = document.getElementById('jobApplicationForm');
    const submitBtn = document.getElementById('submitBtn');
    const successMessage = document.getElementById('successMessage');
    const errorMessage = document.getElementById('errorMessage');
    
    // File upload functionality
    initFileUpload();
    
    // Form submission
    if (form) {
        form.addEventListener('submit', handleFormSubmit);
    }
    
    function initFileUpload() {
        const fileUploadAreas = document.querySelectorAll('.file-upload-area');
        
        fileUploadAreas.forEach(area => {
            const fileInput = area.querySelector('.file-input');
            const filePreview = area.querySelector('.file-preview');
            const fileUploadContent = area.querySelector('.file-upload-content');
            const fileRemove = area.querySelector('.file-remove');
            
            // Click to upload
            area.addEventListener('click', () => {
                fileInput.click();
            });
            
            // File input change
            fileInput.addEventListener('change', (e) => {
                if (e.target.files.length > 0) {
                    handleFileSelect(e.target.files[0], area, filePreview, fileUploadContent);
                }
            });
            
            // Drag and drop
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
                
                const files = e.dataTransfer.files;
                if (files.length > 0) {
                    const file = files[0];
                    if (validateFile(file)) {
                        fileInput.files = files;
                        handleFileSelect(file, area, filePreview, fileUploadContent);
                    }
                }
            });
            
            // Remove file
            if (fileRemove) {
                fileRemove.addEventListener('click', (e) => {
                    e.stopPropagation();
                    resetFileUpload(area, filePreview, fileUploadContent, fileInput);
                });
            }
        });
    }
    
    function validateFile(file) {
        const allowedTypes = ['.pdf', '.doc', '.docx'];
        const maxSize = 10 * 1024 * 1024; // 10MB
        
        const fileExtension = '.' + file.name.split('.').pop().toLowerCase();
        
        if (!allowedTypes.includes(fileExtension)) {
            showError('Lütfen PDF, DOC veya DOCX formatında bir dosya seçin.');
            return false;
        }
        
        if (file.size > maxSize) {
            showError('Dosya boyutu 10MB\'dan büyük olamaz.');
            return false;
        }
        
        return true;
    }
    
    function handleFileSelect(file, area, filePreview, fileUploadContent) {
        if (!validateFile(file)) {
            return;
        }
        
        const fileName = file.name;
        const fileSize = formatFileSize(file.size);
        
        filePreview.querySelector('.file-name').textContent = fileName;
        filePreview.querySelector('.file-size').textContent = fileSize;
        
        fileUploadContent.style.display = 'none';
        filePreview.style.display = 'flex';
    }
    
    function resetFileUpload(area, filePreview, fileUploadContent, fileInput) {
        fileUploadContent.style.display = 'block';
        filePreview.style.display = 'none';
        fileInput.value = '';
    }
    
    function formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }
    
    function handleFormSubmit(e) {
        e.preventDefault();
        
        // Validate form
        if (!validateForm()) {
            return;
        }
        
        // Show loading state
        setLoadingState(true);
        
        // Prepare FormData
        const formData = new FormData();
        
        // Add form fields
        const formFields = [
            'first_name', 'last_name', 'email', 'city', 
            'cover_letter', 'languages', 'work_references', 'kvkk_approval'
        ];
        
        formFields.forEach(field => {
            const input = form.querySelector(`[name="${field}"]`);
            if (input) {
                if (input.type === 'checkbox') {
                    if (input.checked) {
                        formData.append(field, 'true');
                    }
                } else if (input.value.trim()) {
                    formData.append(field, input.value.trim());
                }
            }
        });
        
        // Add CV file
        const cvFile = form.querySelector('#cv_file');
        if (cvFile && cvFile.files.length > 0) {
            formData.append('cv_file', cvFile.files[0]);
        }
        
        // Send AJAX request
        fetch('/backend/make-job-application', {
            method: 'POST',
            body: formData
        })
        .then(response => response.json())
        .then(data => {
            setLoadingState(false);
            
            if (data.status === 201) {
                showSuccess();
                form.reset();
                resetAllFileUploads();
            } else {
                showError(data.message || 'Bir hata oluştu. Lütfen tekrar deneyin.');
            }
        })
        .catch(error => {
            setLoadingState(false);
            console.error('Error:', error);
            showError('Server Hatası: Lütfen daha sonra tekrar deneyin.');
        });
    }
    
    function validateForm() {
        const requiredFields = [
            { name: 'first_name', label: 'Ad' },
            { name: 'last_name', label: 'Soyad' },
            { name: 'email', label: 'E-posta' },
            { name: 'city', label: 'Şehir' },
            { name: 'cover_letter', label: 'Ön Yazı' }
        ];
        
        for (const field of requiredFields) {
            const input = form.querySelector(`[name="${field.name}"]`);
            if (!input || !input.value.trim()) {
                showError(`${field.label} alanı zorunludur.`);
                input?.focus();
                return false;
            }
        }
        
        // Validate email format
        const emailInput = form.querySelector('[name="email"]');
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailRegex.test(emailInput.value)) {
            showError('Lütfen geçerli bir e-posta adresi girin.');
            emailInput.focus();
            return false;
        }
        
        // Validate CV file
        const cvFile = form.querySelector('#cv_file');
        if (!cvFile || !cvFile.files.length) {
            showError('CV dosyası zorunludur.');
            return false;
        }
        
        // Validate KVKK approval
        const kvkkApproval = form.querySelector('#kvkk_approval');
        if (!kvkkApproval || !kvkkApproval.checked) {
            showError('KVKK onayı zorunludur. Lütfen kişisel verilerinizin işlenmesini kabul edin.');
            return false;
        }
        
        return true;
    }
    
    function setLoadingState(loading) {
        if (loading) {
            submitBtn.disabled = true;
            submitBtn.classList.add('btn-loading');
            submitBtn.querySelector('span').textContent = 'Gönderiliyor...';
        } else {
            submitBtn.disabled = false;
            submitBtn.classList.remove('btn-loading');
            submitBtn.querySelector('span').textContent = 'Başvuruyu Gönder';
        }
    }
    
    function showSuccess() {
        hideMessages();
        successMessage.style.display = 'flex';
        successMessage.scrollIntoView({ behavior: 'smooth', block: 'center' });
        
        // Auto hide after 5 seconds
        setTimeout(() => {
            successMessage.style.display = 'none';
        }, 5000);
    }
    
    function showError(message) {
        hideMessages();
        errorMessage.querySelector('span').textContent = message;
        errorMessage.style.display = 'flex';
        errorMessage.scrollIntoView({ behavior: 'smooth', block: 'center' });
        
        // Auto hide after 5 seconds
        setTimeout(() => {
            errorMessage.style.display = 'none';
        }, 5000);
    }
    
    function hideMessages() {
        successMessage.style.display = 'none';
        errorMessage.style.display = 'none';
    }
    
    function resetAllFileUploads() {
        const fileUploadAreas = document.querySelectorAll('.file-upload-area');
        
        fileUploadAreas.forEach(area => {
            const filePreview = area.querySelector('.file-preview');
            const fileUploadContent = area.querySelector('.file-upload-content');
            const fileInput = area.querySelector('.file-input');
            
            resetFileUpload(area, filePreview, fileUploadContent, fileInput);
        });
    }
    
    // Auto-hide messages when clicking outside
    document.addEventListener('click', (e) => {
        if (!successMessage.contains(e.target) && !errorMessage.contains(e.target)) {
            hideMessages();
        }
    });
    
    // Close messages with escape key
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            hideMessages();
        }
    });
});
