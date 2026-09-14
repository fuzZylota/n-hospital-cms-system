// Anlaşmalı Kurumlar Load More Functionality
document.addEventListener('DOMContentLoaded', function() {
    const container = document.getElementById('anlasmaliKurumlarContainer');
    const loadMoreBtn = document.getElementById('anlasmaliKurumlarLoadMoreBtn');
    const loadMoreContainer = document.getElementById('anlasmaliKurumlarLoadMoreContainer');
    
    if (!container || !loadMoreBtn) return;
    
    let isLoading = false;
    let hasMore = true;
    
    // Extract PathOnStart from existing image or script tag
    let pathOnStart = '';
    const existingImage = container.querySelector('.anlasmali-kurum-logo');
    if (existingImage && existingImage.src) {
        const src = existingImage.src;
        const match = src.match(/^(.+?)(?:files\/|assets\/)/);
        if (match) {
            pathOnStart = match[1];
        }
    }
    
    // Fallback: try to get from script tag
    if (!pathOnStart) {
        const scriptTag = document.querySelector('script[src*="anlasmali-kurumlar.js"]');
        if (scriptTag && scriptTag.src) {
            const src = scriptTag.src;
            const match = src.match(/^(.+?)js\/frontend\/anlasmali-kurumlar\.js/);
            if (match) {
                pathOnStart = match[1];
            }
        }
    }
    
    loadMoreBtn.addEventListener('click', async function() {
        if (isLoading || !hasMore) return;
        
        isLoading = true;
        loadMoreBtn.disabled = true;
        loadMoreBtn.querySelector('span').textContent = 'Yükleniyor...';
        
        // Get current count of cards
        const currentCount = container.querySelectorAll('.anlasmali-kurum-item').length;
        
        try {
            const response = await fetch('/backend/get-anlasmali-kurumlar-with-pagination', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    offset: currentCount
                })
            });
            
            const data = await response.json();
            
            if (data && data.status === 200 && data.data && Array.isArray(data.data)) {
                const newKurumlar = data.data;
                
                if (newKurumlar.length === 0) {
                    hasMore = false;
                    loadMoreContainer.style.display = 'none';
                } else {
                    // Append new cards
                    const showPictures = container.querySelector('.anlasmali-kurum-card-header') !== null;
                    
                    newKurumlar.forEach((kurum, index) => {
                        const colDiv = document.createElement('div');
                        colDiv.className = 'col-xl-6 col-lg-12 col-md-6 wow fadeInUp anlasmali-kurum-item';
                        colDiv.setAttribute('data-wow-duration', '1500ms');
                        colDiv.setAttribute('data-wow-delay', (index * 100) + 'ms');
                        
                        let cardHTML = '<div class="anlasmali-kurum-card">';
                        
                        if (showPictures) {
                            cardHTML += '<div class="anlasmali-kurum-card-header">';
                            if (kurum.logo_path && kurum.logo_path !== '') {
                                cardHTML += '<img src="' + pathOnStart + kurum.logo_path + '" alt="' + (kurum.logo_alt_text || kurum.name) + '" title="' + (kurum.logo_title || kurum.name) + '" class="anlasmali-kurum-logo">';
                            } else {
                                cardHTML += '<img src="' + pathOnStart + 'files/defaults/anlasmali-kurumlar/anlasmali-kurumlar-default.png" alt="' + kurum.name + '" title="' + kurum.name + '" class="anlasmali-kurum-logo">';
                            }
                            cardHTML += '</div>';
                        }
                        
                        cardHTML += '<div class="anlasmali-kurum-card-content">';
                        cardHTML += '<h3 class="anlasmali-kurum-name">' + (kurum.name || '') + '</h3>';
                        
                        if (kurum.sube_name && kurum.sube_name !== '') {
                            cardHTML += '<p class="anlasmali-kurum-sube"><i class="fas fa-building"></i> ' + kurum.sube_name + '</p>';
                        }
                        
                        if (kurum.discount_rate && kurum.discount_rate > 0) {
                            cardHTML += '<p class="anlasmali-kurum-discount">İndirim Oranı: %' + kurum.discount_rate + '</p>';
                        }
                        
                        if (kurum.address && kurum.address !== '') {
                            cardHTML += '<p class="anlasmali-kurum-address">' + kurum.address + '</p>';
                        }
                        
                        cardHTML += '</div></div>';
                        
                        colDiv.innerHTML = cardHTML;
                        container.appendChild(colDiv);
                    });
                    
                    // Reinitialize WOW animations for new elements
                    if (typeof WOW !== 'undefined') {
                        new WOW().init();
                    }
                }
            } else {
                hasMore = false;
                loadMoreContainer.style.display = 'none';
            }
        } catch (error) {
            console.error('Error loading more anlasmali kurumlar:', error);
            loadMoreBtn.querySelector('span').textContent = 'Hata oluştu';
        } finally {
            isLoading = false;
            loadMoreBtn.disabled = false;
            if (hasMore) {
                loadMoreBtn.querySelector('span').textContent = 'Devamını Getir';
            }
        }
    });
});

