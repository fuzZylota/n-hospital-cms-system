// Tibbi Birim Düzenle - adapted from sube-duzenle.js

class TibbiBirimEditHandler {
    constructor() {
        this.tbid = window.pageData?.tbid;
        this.generalForm = document.getElementById('generalForm');
        this.pictureForm = document.getElementById('pictureForm');
        this.videoForm = document.getElementById('videoForm');
        this.generalSubmitBtn = document.getElementById('generalSubmitBtn');
        this.pictureSubmitBtn = document.getElementById('pictureSubmitBtn');
        this.videoSubmitBtn = document.getElementById('videoSubmitBtn');
        this.currentDeleteTarget = null;
        this.originalValues = {};
        this.init();
    }

    init() {
        this.initializeOriginalValues();
        this.setupGeneralForm();
        this.setupPictureForm();
        this.setupVideoForm();
        this.setupFileUploads();
        this.setupToggleSwitches();
        this.setupPictureDeletion();
        this.setupVideoDeletion();
        this.setupUrlGeneration();
    }

    initializeOriginalValues() {
        const altTextInput = document.getElementById('old_cover_alt_text');
        const titleInput = document.getElementById('old_cover_title');
        this.originalValues.altText = altTextInput ? altTextInput.value : '';
        this.originalValues.title = titleInput ? titleInput.value : '';
    }

    setupGeneralForm() {
        this.generalForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            if (!this.validateGeneralForm()) return;
            this.setGeneralLoadingState(true);
            try {
                const data = {};
                const allInputs = this.generalForm.querySelectorAll('input, textarea, select');
                allInputs.forEach(input => {
                    if (input.type === 'checkbox') data[input.name] = input.checked;
                    else data[input.name] = input.value;
                });

                const res = await fetch(`/backend/tibbi-birim/${this.tbid}/edit`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(data)
                });
                const result = await res.json();
                if (result.status === 201) {
                    this.showSuccessModal('Tıbbi birim bilgileri başarıyla güncellendi.');
                } else {
                    throw new Error(result.message || 'Güncelleme işlemi başarısız oldu.');
                }
            } catch (err) {
                this.showErrorModal(err.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
            } finally {
                this.setGeneralLoadingState(false);
            }
        });
    }

    setupPictureForm() {
        this.pictureForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            if (!this.validatePictureForm()) return;
            this.setPictureLoadingState(true);
            try {
                const formData = new FormData(this.pictureForm);
                const res = await fetch(`/backend/tibbi-birim/${this.tbid}/update-picture`, { method: 'POST', body: formData });
                const result = await res.json();
                if (result.status === 201) {
                    this.showSuccessModal('Kapak görseli başarıyla güncellendi.');
                    setTimeout(() => window.location.reload(), 1200);
                } else {
                    throw new Error(result.message || 'Görsel güncelleme işlemi başarısız oldu.');
                }
            } catch (err) {
                this.showErrorModal(err.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
            } finally {
                this.setPictureLoadingState(false);
            }
        });
    }

    setupVideoForm() {
        this.videoForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            if (!this.validateVideoForm()) return;
            this.setVideoLoadingState(true);
            try {
                const formData = new FormData(this.videoForm);
                const res = await fetch(`/backend/tibbi-birim/${this.tbid}/update-video`, { method: 'POST', body: formData });
                const result = await res.json();
                if (result.status === 201) {
                    this.showSuccessModal('Tanıtım videosu başarıyla güncellendi.');
                    setTimeout(() => window.location.reload(), 1200);
                } else {
                    throw new Error(result.message || 'Video güncelleme işlemi başarısız oldu.');
                }
            } catch (err) {
                this.showErrorModal(err.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
            } finally {
                this.setVideoLoadingState(false);
            }
        });
    }

    setupFileUploads() {
        const areas = document.querySelectorAll('.file-upload-area');
        areas.forEach(area => {
            const input = area.querySelector('.file-input');
            const content = area.querySelector('.file-upload-content');
            const preview = area.querySelector('.file-preview');
            const previewImage = area.querySelector('.preview-image');
            const removeBtn = area.querySelector('.file-remove');
            const browseLink = area.querySelector('.file-browse');

            browseLink?.addEventListener('click', (e) => { e.preventDefault(); input.click(); });
            area.addEventListener('click', (e) => { if (e.target === area || e.target === content) input.click(); });
            area.addEventListener('dragover', (e) => { e.preventDefault(); area.classList.add('dragover'); });
            area.addEventListener('dragleave', (e) => { e.preventDefault(); if (!area.contains(e.relatedTarget)) area.classList.remove('dragover'); });
            area.addEventListener('drop', (e) => { e.preventDefault(); area.classList.remove('dragover'); const files = e.dataTransfer.files; if (files.length) this.handleFileSelect(input, files[0], content, preview, previewImage, area); });
            input.addEventListener('change', (e) => { if (e.target.files.length) this.handleFileSelect(input, e.target.files[0], content, preview, previewImage, area); });
            removeBtn.addEventListener('click', (e) => { e.stopPropagation(); this.removeFile(input, content, preview, area); });
        });
    }

    handleFileSelect(input, file, content, preview, previewImage, area) {
        const isVideo = input.accept.includes('video/*');
        const allowedImages = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp'];
        const allowedVideos = ['video/mp4', 'video/webm'];
        
        if (isVideo) {
            if (!allowedVideos.includes(file.type)) { 
                this.showErrorModal('Geçersiz video türü. MP4 veya WEBM dosyası yükleyin.'); 
                return; 
            }
            const max = 50 * 1024 * 1024; // 50MB for videos
            if (file.size > max) { 
                this.showErrorModal('Video dosya boyutu 50MB\'dan büyük olamaz.'); 
                return; 
            }
        } else {
            if (!allowedImages.includes(file.type)) { 
                this.showErrorModal('Geçersiz dosya türü. PNG, JPG veya WEBP dosyası yükleyin.'); 
                return; 
            }
            const max = 5 * 1024 * 1024; // 5MB for images
            if (file.size > max) { 
                this.showErrorModal('Dosya boyutu 5MB\'dan büyük olamaz.'); 
                return; 
            }
        }
        
        const dt = new DataTransfer(); 
        dt.items.add(file); 
        input.files = dt.files;
        
        const reader = new FileReader();
        reader.onload = (e) => {
            if (isVideo) {
                const videoPreview = preview.querySelector('.preview-video');
                if (videoPreview) {
                    videoPreview.src = e.target.result;
                }
            } else {
                previewImage.src = e.target.result;
            }
            content.style.display = 'none'; 
            preview.style.display = 'flex'; 
            area.classList.add('has-file');
        };
        reader.readAsDataURL(file);
    }

    removeFile(input, content, preview, area) { input.value = ''; content.style.display = 'block'; preview.style.display = 'none'; area.classList.remove('has-file'); }

    setupToggleSwitches() {
        const toggles = document.querySelectorAll('.toggle-switch input[type="checkbox"]');
        toggles.forEach(toggle => {
            const updateText = () => { const el = toggle.parentNode.querySelector('.toggle-text'); if (el) el.textContent = toggle.checked ? 'Aktif' : 'Pasif'; };
            toggle.addEventListener('change', updateText); updateText();
        });
    }

    setupPictureDeletion() {
        const deleteBtn = document.getElementById('deletePictureBtn');
        const confirmBtn = document.getElementById('confirmDeletePictureBtn');
        if (deleteBtn) {
            deleteBtn.addEventListener('click', () => { this.showDeletePictureModal(); });
        }
        if (confirmBtn) {
            confirmBtn.addEventListener('click', async () => { await this.deletePicture(); });
        }
    }

    setupVideoDeletion() {
        const deleteBtn = document.getElementById('deleteVideoBtn');
        const confirmBtn = document.getElementById('confirmDeleteVideoBtn');
        if (deleteBtn) {
            deleteBtn.addEventListener('click', () => { this.showDeleteVideoModal(); });
        }
        if (confirmBtn) {
            confirmBtn.addEventListener('click', async () => { await this.deleteVideo(); });
        }
    }

    async deletePicture() {
        this.setDeletePictureLoadingState(true);
        try {
            const res = await fetch(`/backend/tibbi-birim/${this.tbid}/delete-picture`, { method: 'POST' });
            const result = await res.json();
            if (result.status === 201) {
                this.showSuccessModal('Kapak görseli başarıyla silindi.');
                setTimeout(() => window.location.reload(), 1000);
            } else {
                throw new Error(result.message || 'Görsel silme işlemi başarısız oldu.');
            }
        } catch (err) {
            this.showErrorModal(err.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
        } finally {
            this.setDeletePictureLoadingState(false);
            this.closeModal('deletePictureModal');
        }
    }

    async deleteVideo() {
        this.setDeleteVideoLoadingState(true);
        try {
            const res = await fetch(`/backend/tibbi-birim/${this.tbid}/delete-video`, { method: 'POST' });
            const result = await res.json();
            if (result.status === 201) {
                this.showSuccessModal('Tanıtım videosu başarıyla silindi.');
                setTimeout(() => window.location.reload(), 1000);
            } else {
                throw new Error(result.message || 'Video silme işlemi başarısız oldu.');
            }
        } catch (err) {
            this.showErrorModal(err.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
        } finally {
            this.setDeleteVideoLoadingState(false);
            this.closeModal('deleteVideoModal');
        }
    }

    setupUrlGeneration() {
        const nameInput = document.getElementById('name');
        const urlInput = document.getElementById('url_name');
        if (nameInput && urlInput) {
            nameInput.addEventListener('input', () => {
                if (!urlInput.value || urlInput.dataset.autoGenerated === 'true') {
                    const urlFriendly = this.generateUrlFriendlyString(nameInput.value);
                    urlInput.value = urlFriendly; urlInput.dataset.autoGenerated = 'true';
                }
            });
            urlInput.addEventListener('input', () => { if (urlInput.value) urlInput.dataset.autoGenerated = 'false'; });
        }
    }

    generateUrlFriendlyString(text) {
        const tr = { 'ç':'c','Ç':'C','ğ':'g','Ğ':'G','ı':'i','I':'I','İ':'I','i':'i','ö':'o','Ö':'O','ş':'s','Ş':'S','ü':'u','Ü':'U' };
        return text.split('').map(ch => tr[ch] || ch).join('').toLowerCase().replace(/[^a-z0-9]+/g,'-').replace(/^-+|-+$/g,'').substring(0,50);
    }

    validateGeneralForm() {
        let ok = true; const req = this.generalForm.querySelectorAll('[required]');
        req.forEach(f => { if (!this.validateField(f)) ok = false; });
        if (!ok) { this.showErrorModal('Lütfen tüm alanları doğru şekilde doldurun.'); const first = this.generalForm.querySelector('.error'); if (first) { first.scrollIntoView({behavior:'smooth', block:'center'}); first.focus(); } }
        return ok;
    }

    validatePictureForm() {
        const fileInput = this.pictureForm.querySelector('input[type="file"]');
        const altTextInput = document.getElementById('cover_alt_text');
        const titleInput = document.getElementById('cover_title');
        const hasFile = fileInput && fileInput.files.length > 0;
        let hasMeta = false;
        if (altTextInput && altTextInput.value !== this.originalValues.altText) hasMeta = true;
        if (titleInput && titleInput.value !== this.originalValues.title) hasMeta = true;
        if (!hasFile && !hasMeta) {
            this.showErrorModal('Lütfen bir dosya seçin veya görsel bilgilerini değiştirin.');
            return false;
        }
        return true;
    }

    validateVideoForm() {
        const fileInput = this.videoForm.querySelector('input[type="file"]');
        const hasFile = fileInput && fileInput.files.length > 0;
        if (!hasFile) {
            this.showErrorModal('Lütfen bir video dosyası seçin.');
            return false;
        }
        return true;
    }

    validateField(field) {
        const value = field.value.trim(); let valid = true; let msg = '';
        if (field.hasAttribute('required') && !value) { valid = false; msg = 'Bu alan zorunludur.'; }
        this.setFieldState(field, valid, msg); return valid;
    }

    setFieldState(field, isValid, msg) {
        const group = field.closest('.form-group'); let el = group.querySelector('.field-error'); if (el) el.remove();
        field.classList.remove('error','success'); field.classList.add(isValid?'success':'error');
        if (!isValid && msg) { el = document.createElement('div'); el.className='field-error'; el.innerHTML = `<i class="icon-alert-circle"></i> ${msg}`; group.appendChild(el); }
    }

    setGeneralLoadingState(loading) { if (loading) { this.generalSubmitBtn.disabled = true; this.generalSubmitBtn.querySelector('.btn-text').style.opacity='0'; this.generalSubmitBtn.querySelector('.btn-loader').style.display='block'; this.generalForm.classList.add('form-loading'); } else { this.generalSubmitBtn.disabled=false; this.generalSubmitBtn.querySelector('.btn-text').style.opacity='1'; this.generalSubmitBtn.querySelector('.btn-loader').style.display='none'; this.generalForm.classList.remove('form-loading'); } }
    setPictureLoadingState(loading) { if (loading) { this.pictureSubmitBtn.disabled = true; this.pictureSubmitBtn.querySelector('.btn-text').style.opacity='0'; this.pictureSubmitBtn.querySelector('.btn-loader').style.display='block'; this.pictureForm.classList.add('form-loading'); } else { this.pictureSubmitBtn.disabled=false; this.pictureSubmitBtn.querySelector('.btn-text').style.opacity='1'; this.pictureSubmitBtn.querySelector('.btn-loader').style.display='none'; this.pictureForm.classList.remove('form-loading'); } }
    setVideoLoadingState(loading) { if (loading) { this.videoSubmitBtn.disabled = true; this.videoSubmitBtn.querySelector('.btn-text').style.opacity='0'; this.videoSubmitBtn.querySelector('.btn-loader').style.display='block'; this.videoForm.classList.add('form-loading'); } else { this.videoSubmitBtn.disabled=false; this.videoSubmitBtn.querySelector('.btn-text').style.opacity='1'; this.videoSubmitBtn.querySelector('.btn-loader').style.display='none'; this.videoForm.classList.remove('form-loading'); } }
    setDeletePictureLoadingState(loading) { const btn = document.getElementById('confirmDeletePictureBtn'); if (!btn) return; if (loading) { btn.disabled=true; btn.querySelector('.btn-text').style.opacity='0'; btn.querySelector('.btn-loader').style.display='block'; } else { btn.disabled=false; btn.querySelector('.btn-text').style.opacity='1'; btn.querySelector('.btn-loader').style.display='none'; } }
    setDeleteVideoLoadingState(loading) { const btn = document.getElementById('confirmDeleteVideoBtn'); if (!btn) return; if (loading) { btn.disabled=true; btn.querySelector('.btn-text').style.opacity='0'; btn.querySelector('.btn-loader').style.display='block'; } else { btn.disabled=false; btn.querySelector('.btn-text').style.opacity='1'; btn.querySelector('.btn-loader').style.display='none'; } }

    showSuccessModal(message) { const modal = document.getElementById('successModal'); const msg = document.getElementById('successMessage'); if (msg) msg.textContent = message; modal.style.display='flex'; setTimeout(()=>modal.classList.add('show'),10); }
    showErrorModal(message) { const modal = document.getElementById('errorModal'); const msg = document.getElementById('errorMessage'); if (msg) msg.textContent = message; modal.style.display='flex'; setTimeout(()=>modal.classList.add('show'),10); }
    showDeletePictureModal() { const modal = document.getElementById('deletePictureModal'); modal.style.display='flex'; setTimeout(()=>modal.classList.add('show'),10); }
    showDeleteVideoModal() { const modal = document.getElementById('deleteVideoModal'); modal.style.display='flex'; setTimeout(()=>modal.classList.add('show'),10); }
    closeModal(modalId) { const modal = document.getElementById(modalId); modal.classList.remove('show'); setTimeout(()=>{ modal.style.display='none'; },300); }
}

function closeModal(modalId){ const modal = document.getElementById(modalId); modal.classList.remove('show'); setTimeout(()=>{ modal.style.display='none'; },300); }

document.addEventListener('DOMContentLoaded', ()=>{
    new TibbiBirimEditHandler();
    document.addEventListener('click',(e)=>{ if (e.target.classList.contains('modal')) closeModal(e.target.id); });
    document.addEventListener('keydown',(e)=>{ if (e.key==='Escape'){ const m=document.querySelector('.modal.show'); if (m) closeModal(m.id); }});
});

window.TibbiBirimEditHandler = TibbiBirimEditHandler;

