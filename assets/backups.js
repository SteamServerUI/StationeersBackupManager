document.addEventListener('DOMContentLoaded', () => {

        fetchBackups();
});


function fetchBackups() {
    const limit = document.getElementById('backupLimit').value;
    const url = limit ? `/plugins/StationeersBackupManager/api/v1/backups?limit=${limit}` : '/plugins/StationeersBackupManager/api/v1/backups';
    
    return fetch(url)
        .then(response => {
            const contentType = response.headers.get('Content-Type');
            if (contentType && contentType.includes('application/json')) {
                return response.json().then(data => ({ status: response.ok, data }));
            } else {
                return response.text().then(text => ({ status: response.ok, text }));
            }
        })
        .then(result => {
            const backupList = document.getElementById('backupList');
            backupList.innerHTML = '';
            
            if (!result.status || result.text) {
                backupList.innerHTML = `<li class="backuperror">${result.text || 'Failed to load backups'}</li>`;
                return;
            }
            
            const data = result.data;
            if (!data || data.length === 0) {
                backupList.innerHTML = '<li class="no-backups">No valid backup files found.</li>';
                return;
            }
            
            let animationCount = 0;
            data.forEach((backup) => {
                const li = document.createElement('li');
                li.className = 'backup-item';
                
                const backupType = getBackupType(backup);
                const fileName = "Backup Index: " + backup.Index;
                const formattedDate = "Created: " + new Date(backup.ModTime).toLocaleString();
                
                li.innerHTML = `
                    <div class="backup-info">
                        <div class="backup-header">
                            <span class="backup-name">${fileName}</span>
                            <span class="backup-type ${backupType.toLowerCase()}">${backupType}</span>
                        </div>
                        <div class="backup-date">${formattedDate}</div>
                    </div>
                    <button class="restore-btn" onclick="restoreBackup(${backup.Index})">Restore</button>
                `;
                
                backupList.appendChild(li);
                
                if (animationCount < 20) {
                    setTimeout(() => {
                        li.classList.add('animate-in');
                    }, animationCount * 50);
                    animationCount++;
                }
            });
        })
        .catch(err => {
            console.error("Failed to fetch backups:", err);
            document.getElementById('backupList').innerHTML = '<li class="backuperror">Failed to load backups</li>';
        });
}

function getBackupType(backup) {
    if (backup.BinFile && backup.XMLFile && backup.MetaFile) {
        return 'preterrain-trio';
    } else if (backup.BinFile && !backup.XMLFile && !backup.MetaFile) {
        return 'Dotsave';
    }
    return 'Unknown';
}

function extractIndex(backupText) {
    return backupText.match(/Index: (\d+)/)?.[1] || null;
}

function restoreBackup(index) {
    const status = document.getElementById('status');
    fetch(`/plugins/StationeersBackupManager/api/v1/backups/restore?index=${index}`)
        .then(response => response.text())
        .then(data => {
            status.hidden = false;
            typeTextWithCallback(status, data, 20, () => {
                setTimeout(() => status.hidden = true, 30000);
            });
        })
        .catch(err => console.error(`Failed to restore backup ${index}:`, err));
}

// Utility function for typing text with a callback
function typeTextWithCallback(element, text, speed, callback) {
    if (element.dataset.isTyping === 'true') {
        clearTimeout(element.dataset.timeoutId);
    }

    element.textContent = '';
    element.dataset.isTyping = 'true';
    let i = 0;
    
    const typeChar = () => {
        if (i < text.length) {
            element.textContent += text.charAt(i++);
            const timeoutId = setTimeout(typeChar, speed);
            element.dataset.timeoutId = timeoutId;
        } else {
            element.dataset.isTyping = 'false';
            delete element.dataset.timeoutId;
            if (callback) setTimeout(callback, 50);
        }
    };
    typeChar();
}