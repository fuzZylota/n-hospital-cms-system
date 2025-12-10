let recaptchaVerifiedFast = false;

function isFastRandevuMobileDevice() {
    if (typeof navigator === 'undefined') return false;
    const ua = navigator.userAgent || '';
    return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(ua);
}

function updateFastRandevuSubmitState() {
    const kvkkCheckbox = document.getElementById('fastRandevuKVKK');
    const submitButton = document.getElementById('fastRandevuSubmit');
    const recaptchaWidget = document.querySelector('#fastRandevu .g-recaptcha');

    if (!submitButton) {
        return;
    }

    let shouldDisable = !kvkkCheckbox || !kvkkCheckbox.checked;
    if (recaptchaWidget) {
        shouldDisable = shouldDisable || !recaptchaVerifiedFast;
    }

    submitButton.disabled = shouldDisable;
}

function setFastRandevuFeedback(type, message) {
    const feedback = document.getElementById('fastRandevuFeedback');
    if (!feedback) return;

    feedback.className = 'fast-randevu__feedback';

    if (!type || !message) {
        feedback.textContent = '';
        return;
    }

    feedback.classList.add('is-visible');
    if (type === 'loading') {
        feedback.classList.add('fast-randevu__feedback--loading');
    } else if (type === 'success') {
        feedback.classList.add('fast-randevu__feedback--success');
    } else if (type === 'error') {
        feedback.classList.add('fast-randevu__feedback--error');
    }
    feedback.textContent = message;
}

window.onRecaptchaVerifiedFast = function(token) {
    recaptchaVerifiedFast = true;
    const tokenInput = document.getElementById('fastRandevuRecaptchaToken');
    if (tokenInput) {
        tokenInput.value = token;
    }
    updateFastRandevuSubmitState();
};

window.onRecaptchaExpiredFast = function() {
    recaptchaVerifiedFast = false;
    const tokenInput = document.getElementById('fastRandevuRecaptchaToken');
    if (tokenInput) {
        tokenInput.value = '';
    }
    updateFastRandevuSubmitState();
};

async function loadSubeler() {
    const subeSelect = document.getElementById('fastRandevuSube');
    if (!subeSelect) return;

    try {
        const response = await fetch('/backend/get-all-subeler', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });

        const data = await response.json();
        if (data && data.status === 200 && data.data) {
            subeSelect.innerHTML = '<option value="">Şube Seçiniz</option>';
            data.data.forEach(sube => {
                const option = document.createElement('option');
                option.value = sube.sid;
                option.textContent = sube.name;
                subeSelect.appendChild(option);
            });
        }
    } catch (error) {
        console.error('Şubeler yüklenirken hata oluştu:', error);
    }
}

function setFastRandevuOpenState(isOpen) {
    const container = document.getElementById('fastRandevu');
    const toggleButton = document.getElementById('fastRandevuToggle');
    const panel = container ? container.querySelector('.fast-randevu__panel') : null;

    if (!container || !toggleButton || !panel) return;

    if (isOpen) {
        container.classList.add('is-open');
        loadSubeler();
    } else {
        container.classList.remove('is-open');
    }
    toggleButton.setAttribute('aria-expanded', isOpen ? 'true' : 'false');
    panel.setAttribute('aria-hidden', isOpen ? 'false' : 'true');

    logFastRandevuState(isOpen ? 'open' : 'close');
}

function handleFastRandevuOutsideInteraction(event) {
    const container = document.getElementById('fastRandevu');
    if (!container || !container.classList.contains('is-open')) return;

    const panel = container.querySelector('.fast-randevu__panel');
    const toggleButton = document.getElementById('fastRandevuToggle');
    const closeButton = document.getElementById('fastRandevuClose');
    const mobileLineButton = document.getElementById('fastRandevuMobileLine');

    const target = event.target;

    // Ignore clicks/taps inside panel or on known controls
    if (panel && panel.contains(target)) return;
    if (toggleButton && toggleButton.contains(target)) return;
    if (closeButton && closeButton.contains(target)) return;
    if (mobileLineButton && mobileLineButton.contains(target)) return;

    setFastRandevuOpenState(false);
}

function logFastRandevuState(source = 'manual') {
    const container = document.getElementById('fastRandevu');
    const mobileLine = document.getElementById('fastRandevuMobileLine');
    const panel = container ? container.querySelector('.fast-randevu__panel') : null;
    const toggleButton = document.getElementById('fastRandevuToggle');

    const logPayload = {
        source,
        viewport: {
            width: window.innerWidth,
            height: window.innerHeight,
        },
        hasContainer: !!container,
        containerClasses: container ? Array.from(container.classList) : [],
        isOpen: container ? container.classList.contains('is-open') : false,
        toggleVisible: toggleButton ? window.getComputedStyle(toggleButton).display : 'missing',
        panelStyles: panel
            ? {
                  display: window.getComputedStyle(panel).display,
                  opacity: window.getComputedStyle(panel).opacity,
                  transform: window.getComputedStyle(panel).transform,
              }
            : 'missing',
        mobileLine: mobileLine
            ? {
                  display: window.getComputedStyle(mobileLine).display,
                  opacity: window.getComputedStyle(mobileLine).opacity,
                  visibility: window.getComputedStyle(mobileLine).visibility,
                  rect: mobileLine.getBoundingClientRect(),
                  zIndex: window.getComputedStyle(mobileLine).zIndex,
              }
            : 'missing',
    };

    console.log('[FastRandevu Debug]', logPayload);
}

document.addEventListener('DOMContentLoaded', () => {
    const container = document.getElementById('fastRandevu');
    const toggleButton = document.getElementById('fastRandevuToggle');
    const form = document.getElementById('fastRandevuForm');
    const kvkkCheckbox = document.getElementById('fastRandevuKVKK');
    const closeButton = document.getElementById('fastRandevuClose');
    let mobileLineButton = document.getElementById('fastRandevuMobileLine');

    if (!mobileLineButton && container) {
        mobileLineButton = document.createElement('button');
        mobileLineButton.type = 'button';
        mobileLineButton.id = 'fastRandevuMobileLine';
        mobileLineButton.className = 'fast-randevu__mobile-line';
        mobileLineButton.setAttribute('aria-label', 'Hızlı randevu panelini aç');
        mobileLineButton.innerHTML = `
            <span class="fast-randevu__mobile-line-label">
                <i class="icon-phone-call"></i>
                Hızlı Randevu
            </span>
            <span class="fast-randevu__mobile-line-badge">7/24</span>
        `;
        container.appendChild(mobileLineButton);
        logFastRandevuState('mobile-line-created');
    }

    if (kvkkCheckbox) {
        kvkkCheckbox.addEventListener('change', updateFastRandevuSubmitState);
    }
    updateFastRandevuSubmitState();

    if (container && toggleButton) {
        toggleButton.addEventListener('click', (event) => {
            event.preventDefault();
            const isOpen = container.classList.contains('is-open');
            setFastRandevuOpenState(!isOpen);
        });
        logFastRandevuState('toggle-mounted');
    }

    if (closeButton) {
        closeButton.addEventListener('click', (event) => {
            event.preventDefault();
            setFastRandevuOpenState(false);
        });
    }

    if (mobileLineButton) {
        mobileLineButton.addEventListener('click', (event) => {
            event.preventDefault();
            setFastRandevuOpenState(true);
            logFastRandevuState('mobile-line-click');
        });
        logFastRandevuState('mobile-line-mounted');
    } else {
        logFastRandevuState('mobile-line-missing');
    }

    logFastRandevuState('dom-ready');

    let resizeTimeout;
    window.addEventListener('resize', () => {
        clearTimeout(resizeTimeout);
        resizeTimeout = setTimeout(() => logFastRandevuState('resize'), 200);
    });

    // Close when clicking/touching outside the panel
    document.addEventListener('click', handleFastRandevuOutsideInteraction);
    document.addEventListener('touchstart', handleFastRandevuOutsideInteraction, { passive: true });

    window.fastRandevuLogState = () => logFastRandevuState('manual-call');

    if (!form) {
        return;
    }

    form.addEventListener('submit', async (event) => {
        event.preventDefault();

        const firstNameInput = document.getElementById('fastRandevuName');
        const lastNameInput = document.getElementById('fastRandevuSurname');
        const phoneInput = document.getElementById('fastRandevuPhone');
        const subeSelect = document.getElementById('fastRandevuSube');
        const recaptchaWidget = document.querySelector('#fastRandevu .g-recaptcha');
        const tokenInput = document.getElementById('fastRandevuRecaptchaToken');
        const submitButton = document.getElementById('fastRandevuSubmit');

        if (!firstNameInput.value.trim() || !lastNameInput.value.trim() || !phoneInput.value.trim()) {
            setFastRandevuFeedback('error', 'Lütfen ad, soyad ve telefon alanlarını doldurun.');
            return;
        }

        if (kvkkCheckbox && !kvkkCheckbox.checked) {
            setFastRandevuFeedback('error', 'KVKK onayı olmadan talep gönderemezsiniz.');
            return;
        }

        let recaptchaToken = null;
        if (recaptchaWidget) {
            if (typeof grecaptcha === 'undefined') {
                setFastRandevuFeedback('error', 'Güvenlik doğrulaması yükleniyor, lütfen bekleyin.');
                return;
            }

            // First check the flag and token input (set by callback)
            if (recaptchaVerifiedFast && tokenInput && tokenInput.value && tokenInput.value !== '') {
                recaptchaToken = tokenInput.value;
            } else {
                // Fallback: try to get response directly from the widget
                // Find the widget ID by checking the reCAPTCHA container
                try {
                    const sitekey = recaptchaWidget.getAttribute('data-sitekey');
                    if (sitekey) {
                        // Try to find the widget by iterating through all widgets
                        // or get response without widget ID (works if only one widget on page)
                        const response = grecaptcha.getResponse();
                        if (response && response !== '') {
                            recaptchaToken = response;
                            // Also update our flag and token input for consistency
                            recaptchaVerifiedFast = true;
                            if (tokenInput) {
                                tokenInput.value = response;
                            }
                        }
                    }
                } catch (e) {
                    // Ignore errors
                }
            }

            if (!recaptchaToken || recaptchaToken === '') {
                setFastRandevuFeedback('error', 'Lütfen güvenlik doğrulamasını tamamlayın.');
                return;
            }
            
            // Ensure token input is updated with the validated token
            if (tokenInput) {
                tokenInput.value = recaptchaToken;
            }
        }

        const payload = {
            patient_first_name: firstNameInput.value.trim(),
            patient_last_name: lastNameInput.value.trim(),
            patient_phone: phoneInput.value.trim(),
            message: 'Hızlı randevu formu üzerinden iletilmiştir.'
        };

        if (subeSelect && subeSelect.value) {
            payload.sid = subeSelect.value;
        }

        if (recaptchaToken) {
            payload.recaptcha_token = recaptchaToken;
        }

        setFastRandevuFeedback('loading', 'Talebiniz gönderiliyor...');
        submitButton.disabled = true;

        try {
            const response = await fetch('/backend/add-randevu-request', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            const data = await response.json();
            if (data && (data.status === 200 || data.status === 201)) {
                setFastRandevuFeedback('success', 'Talebiniz alınmıştır. En kısa sürede tarafınıza dönüş yapılacaktır.');

                try {
                    const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws';
                    const url = `${scheme}://${window.location.host}/backend/notifications`;
                    const ws = new WebSocket(url, 'randevu');
                    payload.rrid = data.rrid;
                    
                    let messageSent = false;
                    const sendMessage = () => {
                        if (messageSent) return;
                        messageSent = true;
                        try {
                            ws.send(JSON.stringify({
                                uid: null,
                                message: JSON.stringify(payload)
                            }));
                            setTimeout(() => {
                                try {
                                    if (ws.readyState === WebSocket.OPEN) {
                                        ws.close();
                                    }
                                } catch (e) {
                                    // Ignore close errors
                                }
                            }, 500);
                        } catch (e) {
                            console.error('WebSocket send error:', e);
                        }
                    };
                    
                    ws.addEventListener('open', sendMessage);
                    ws.addEventListener('error', (error) => {
                        console.error('WebSocket error:', error);
                    });
                    
                    // Timeout fallback
                    setTimeout(() => {
                        if (!messageSent && ws.readyState === WebSocket.OPEN) {
                            sendMessage();
                        }
                    }, 1000);
                } catch (error) {
                    console.error('WebSocket connection error:', error);
                }

                form.reset();
                if (typeof grecaptcha !== 'undefined') {
                    grecaptcha.reset();
                }
                if (tokenInput) {
                    tokenInput.value = '';
                }
                recaptchaVerifiedFast = false;
                updateFastRandevuSubmitState();
            } else {
                setFastRandevuFeedback('error', data?.message || 'Sunucu Hatası: Lütfen daha sonra tekrar deneyin.');
            }
        } catch (error) {
            setFastRandevuFeedback('error', 'Sunucuya bağlanırken bir hata oluştu. Lütfen daha sonra tekrar deneyin.');
        } finally {
            submitButton.disabled = false;
            updateFastRandevuSubmitState();
        }
    });
});

