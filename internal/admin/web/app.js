// AI Prompt Proxy 管理后台 JavaScript

class AIPromptProxyAdmin {
    constructor() {
        this.baseURL = 'http://localhost:8081/api/v1';
        this.models = [];
        this.filteredModels = [];
        this.currentEditingModel = null;
        this.init();
    }

    init() {
        this.bindEvents();
        this.loadModels();
        this.loadStatus();
        
        // 定期刷新状态
        setInterval(() => this.loadStatus(), 30000);
        
        // 添加工具提示功能
        this.initTooltips();
    }

    initTooltips() {
        // 简单的工具提示实现
        document.addEventListener('mouseover', (e) => {
            if (e.target.hasAttribute('data-tooltip')) {
                const tooltip = document.createElement('div');
                tooltip.className = 'fixed bg-gray-800 text-white text-xs px-2 py-1 rounded shadow-lg z-50 pointer-events-none';
                tooltip.textContent = e.target.getAttribute('data-tooltip');
                tooltip.id = 'tooltip';
                
                document.body.appendChild(tooltip);
                
                const rect = e.target.getBoundingClientRect();
                tooltip.style.left = rect.left + (rect.width / 2) - (tooltip.offsetWidth / 2) + 'px';
                tooltip.style.top = rect.top - tooltip.offsetHeight - 5 + 'px';
            }
        });
        
        document.addEventListener('mouseout', (e) => {
            if (e.target.hasAttribute('data-tooltip')) {
                const tooltip = document.getElementById('tooltip');
                if (tooltip) {
                    tooltip.remove();
                }
            }
        });
    }

    bindEvents() {
        // 添加模型按钮 - 支持多个按钮
        document.getElementById('add-model').addEventListener('click', () => {
            this.openModal();
        });

        // 导航栏添加模型按钮
        const addModelNav = document.getElementById('add-model-nav');
        if (addModelNav) {
            addModelNav.addEventListener('click', () => {
                this.openModal();
            });
        }

        // 模态框关闭
        document.getElementById('close-modal').addEventListener('click', () => {
            this.closeModal();
        });
        document.getElementById('cancel-modal').addEventListener('click', () => {
            this.closeModal();
        });

        // 删除模态框
        document.getElementById('cancel-delete').addEventListener('click', () => {
            this.closeDeleteModal();
        });
        document.getElementById('confirm-delete').addEventListener('click', () => {
            this.confirmDelete();
        });

        // 表单提交
        document.getElementById('model-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.saveModel();
        });

        // 搜索和筛选
        document.getElementById('search-input').addEventListener('input', (e) => {
            this.filterModels();
        });
        document.getElementById('type-filter').addEventListener('change', (e) => {
            this.filterModels();
        });

        // 排序功能
        const sortFilter = document.getElementById('sort-filter');
        if (sortFilter) {
            sortFilter.addEventListener('change', () => {
                this.filterModels();
            });
        }

        // 重新加载配置
        document.getElementById('reload-config').addEventListener('click', () => {
            this.reloadConfig();
        });

        // 模型类型变化时的处理
        document.getElementById('model-type').addEventListener('change', (e) => {
            this.handleModelTypeChange(e.target.value);
        });

        // 点击模态框外部关闭
        document.getElementById('model-modal').addEventListener('click', (e) => {
            if (e.target.id === 'model-modal') {
                this.closeModal();
            }
        });
        document.getElementById('delete-modal').addEventListener('click', (e) => {
            if (e.target.id === 'delete-modal') {
                this.closeDeleteModal();
            }
        });

        // 高级配置抽屉展开/收起
        document.addEventListener('click', (e) => {
            if (e.target.id === 'toggle-advanced-prompt' || e.target.closest('#toggle-advanced-prompt')) {
                this.toggleAdvancedPromptConfig();
            }
        });
    }

    async apiRequest(endpoint, options = {}) {
        try {
            const response = await fetch(`${this.baseURL}${endpoint}`, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            console.error('API请求失败:', error);
            this.showToast(`请求失败: ${error.message}`, 'error');
            throw error;
        }
    }

    async loadModels() {
        try {
            const response = await this.apiRequest('/models');
            this.models = response.data.models || [];
            this.filteredModels = [...this.models];
            this.renderModels();
            this.updateTotalCount();
            this.updateModelCounts();
        } catch (error) {
            console.error('加载模型失败:', error);
        }
    }

    async loadStatus() {
        try {
            const response = await this.apiRequest('/config/status');
            const status = response.data;
            
            document.getElementById('service-status').textContent = 
                status.status === 'running' ? '运行中' : '已停止';
            document.getElementById('total-models').textContent = status.total_models;
            
            // 更新状态指示器
            const statusBadge = document.querySelector('.status-badge');
            if (status.status === 'running') {
                statusBadge.className = 'status-badge inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800';
            } else {
                statusBadge.className = 'status-badge inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800';
            }
        } catch (error) {
            console.error('加载状态失败:', error);
        }
    }

    filterModels() {
        const searchTerm = document.getElementById('search-input').value.toLowerCase();
        const typeFilter = document.getElementById('type-filter').value;
        const sortFilter = document.getElementById('sort-filter')?.value || 'updated_at';

        let filteredModels = this.models.filter(model => {
            const matchesSearch = !searchTerm || 
                model.id.toLowerCase().includes(searchTerm) ||
                model.name.toLowerCase().includes(searchTerm) ||
                (model.prompt && model.prompt.toLowerCase().includes(searchTerm));
            
            const matchesType = !typeFilter || model.type === typeFilter;
            
            return matchesSearch && matchesType;
        });

        // 排序
        filteredModels.sort((a, b) => {
            switch (sortFilter) {
                case 'updated_at':
                    // 按更新时间排序（最新的在前）
                    const dateA = new Date(a.updated_at || 0);
                    const dateB = new Date(b.updated_at || 0);
                    return dateB - dateA;
                case 'type':
                    return a.type.localeCompare(b.type);
                case 'recent':
                    // 假设按ID排序作为最近添加的指标
                    return b.id.localeCompare(a.id);
                case 'name':
                default:
                    return a.name.localeCompare(b.name);
            }
        });

        this.filteredModels = filteredModels;
        this.renderModels();
    }

    renderModels() {
        const container = document.getElementById('models-container');
        const emptyState = document.getElementById('empty-state');

        if (this.filteredModels.length === 0) {
            container.innerHTML = '';
            emptyState.classList.remove('hidden');
            return;
        }

        emptyState.classList.add('hidden');
        container.innerHTML = this.filteredModels.map(model => `
            <div class="model-card card-hover bg-white/80 backdrop-blur-sm rounded-2xl shadow-xl p-6 border border-white/20 hover:shadow-2xl transition-all duration-300 hover:scale-105 animate-slide-up">
                <div class="flex justify-between items-start mb-4">
                    <div class="flex items-start space-x-3 flex-1">
                        <div class="w-12 h-12 ${this.getTypeGradient(model.type)} rounded-xl flex items-center justify-center shadow-lg flex-shrink-0">
                            ${this.getTypeIcon(model.type)}
                        </div>
                        <div class="flex-1 min-w-0">
                            <h3 class="text-lg font-bold text-gray-900">${this.escapeHtml(model.name)}</h3>
                            <div class="flex items-center space-x-2">
                                <p class="text-sm text-gray-500 font-medium">${this.escapeHtml(model.id)}</p>
                                <button onclick="app.copyToClipboard('${this.escapeHtml(model.id)}', '模型ID')" class="tooltip text-gray-400 hover:text-blue-500 transition-colors duration-200" data-tooltip="复制模型ID">
                                    <i class="fas fa-copy text-xs"></i>
                                </button>
                            </div>
                            ${model.updated_at ? `<p class="text-xs text-gray-400 mt-1">更新时间: ${this.formatDateTime(model.updated_at)}</p>` : ''}
                        </div>
                    </div>
                    <span class="type-badge-${model.type} px-3 py-1.5 rounded-full text-xs font-semibold shadow-sm flex-shrink-0 ml-3">
                        ${this.getTypeEmoji(model.type)} ${this.getTypeLabel(model.type)}
                    </span>
                </div>
                
                <div class="space-y-3 mb-6">
                    <div class="flex items-center text-sm text-gray-600">
                        <div class="w-8 h-8 bg-gray-100 rounded-lg flex items-center justify-center mr-3">
                            <i class="fas fa-bullseye text-gray-500"></i>
                        </div>
                        <div>
                            <span class="font-semibold text-gray-700">目标:</span>
                            <span class="ml-2 text-gray-900">${this.escapeHtml(model.target)}</span>
                        </div>
                    </div>
                    <div class="flex items-start text-sm text-gray-600">
                        <div class="w-8 h-8 bg-gray-100 rounded-lg flex items-center justify-center mr-3 mt-0.5">
                            <i class="fas fa-link text-gray-500"></i>
                        </div>
                        <div class="flex-1">
                            <span class="font-semibold text-gray-700">URL:</span>
                            <p class="text-gray-900 break-all mt-1">${this.escapeHtml(model.url)}</p>
                        </div>
                    </div>
                    ${model.prompt ? `
                        <div class="flex items-start text-sm text-gray-600">
                            <div class="w-8 h-8 bg-gray-100 rounded-lg flex items-center justify-center mr-3 mt-0.5">
                                <i class="fas fa-comment-dots text-gray-500"></i>
                            </div>
                            <div class="flex-1">
                                <span class="font-semibold text-gray-700">Prompt:</span>
                                <p class="text-gray-600 mt-1 line-clamp-2">${this.escapeHtml(model.prompt || '')}</p>
                            </div>
                        </div>
                    ` : ''}
                </div>
                
                <div class="quick-actions flex justify-end space-x-3 pt-4 border-t border-gray-100">
                    <button onclick="app.editModel('${model.id}')" class="tooltip px-4 py-2.5 text-sm bg-blue-100 text-blue-700 rounded-xl hover:bg-blue-200 transition-all duration-300 hover:scale-105 font-semibold" data-tooltip="编辑模型">
                        <i class="fas fa-edit mr-2"></i>
                        编辑
                    </button>
                    <button onclick="app.deleteModel('${model.id}', '${this.escapeHtml(model.name)}')" class="tooltip px-4 py-2.5 text-sm bg-red-100 text-red-700 rounded-xl hover:bg-red-200 transition-all duration-300 hover:scale-105 font-semibold" data-tooltip="删除模型">
                        <i class="fas fa-trash mr-2"></i>
                        删除
                    </button>
                </div>
            </div>
        `).join('');
    }

    getTypeColor(type) {
        const colors = {
            'chat': 'bg-blue-100 text-blue-800',
            'image': 'bg-green-100 text-green-800',
            'audio': 'bg-purple-100 text-purple-800',
            'video': 'bg-orange-100 text-orange-800'
        };
        return colors[type] || 'bg-gray-100 text-gray-800';
    }

    getTypeGradient(type) {
        const gradients = {
            'chat': 'bg-gradient-to-br from-blue-400 to-blue-600',
            'image': 'bg-gradient-to-br from-green-400 to-green-600',
            'audio': 'bg-gradient-to-br from-purple-400 to-purple-600',
            'video': 'bg-gradient-to-br from-orange-400 to-orange-600'
        };
        return gradients[type] || 'bg-gradient-to-br from-gray-400 to-gray-600';
    }

    getTypeIcon(type) {
        const icons = {
            'chat': '<i class="fas fa-comments text-white"></i>',
            'image': '<i class="fas fa-image text-white"></i>',
            'audio': '<i class="fas fa-volume-up text-white"></i>',
            'video': '<i class="fas fa-video text-white"></i>'
        };
        return icons[type] || '<i class="fas fa-cog text-white"></i>';
    }

    getTypeEmoji(type) {
        const emojis = {
            'chat': '💬',
            'image': '🖼️',
            'audio': '🔊',
            'video': '🎬'
        };
        return emojis[type] || '⚙️';
    }

    getTypeLabel(type) {
        const labels = {
            'chat': '对话',
            'image': '图像',
            'audio': '音频',
            'video': '视频'
        };
        return labels[type] || '其他';
    }

    openModal(model = null) {
        const modal = document.getElementById('model-modal');
        const title = document.getElementById('modal-title');
        const form = document.getElementById('model-form');

        if (model) {
            this.currentEditingModel = model;
            title.textContent = '编辑模型配置';
            this.fillForm(model);
            document.getElementById('model-id').disabled = true;
        } else {
            this.currentEditingModel = null;
            title.textContent = '添加模型配置';
            form.reset();
            document.getElementById('model-id').disabled = false;
            
            // 只设置模型类型的默认值，其他字段保持空白
            document.getElementById('model-type').value = 'chat';
        }

        modal.classList.remove('hidden');
        document.body.style.overflow = 'hidden';
    }

    closeModal() {
        const modal = document.getElementById('model-modal');
        modal.classList.add('hidden');
        document.body.style.overflow = 'auto';
        this.currentEditingModel = null;
    }

    handleModelTypeChange(modelType) {
        // 移除自动填充默认值的逻辑，保持用户的选择
        // 用户可以根据需要手动填写或保持空白
    }

    fillForm(model) {
        document.getElementById('model-id').value = model.id;
        document.getElementById('model-name').value = model.name;
        document.getElementById('model-target').value = model.target;
        document.getElementById('model-prompt').value = model.prompt || '';
        document.getElementById('model-type').value = model.type;
        document.getElementById('model-url').value = model.url;
        
        // 对于可选字段，只有在有值时才填充，否则保持空白
        document.getElementById('model-prompt-path').value = model.prompt_path || '';
        document.getElementById('model-prompt-value-type').value = model.prompt_value_type || '';
        
        // 对于prompt_value，只有在有值时才填充
        const promptValueInput = document.getElementById('model-prompt-value');
        if (model.prompt_value) {
            promptValueInput.value = 
                typeof model.prompt_value === 'string' ? 
                model.prompt_value : 
                JSON.stringify(model.prompt_value, null, 2);
        } else {
            promptValueInput.value = '';
        }
    }

    async saveModel() {
        const form = document.getElementById('model-form');
        const formData = new FormData(form);
        const data = {};

        // 定义所有可能的字段，包括可选字段
        const allFields = [
            'id', 'name', 'target', 'type', 'url', 'prompt', 
            'prompt_path', 'prompt_value_type', 'prompt_value'
        ];

        // 处理所有字段，包括空值
        for (const field of allFields) {
            const value = formData.get(field);
            if (value !== null) {
                data[field] = value.trim();
            }
        }

        // 对于编辑模式，如果模型ID字段被禁用，手动获取其值
        if (this.currentEditingModel) {
            const modelIdInput = document.getElementById('model-id');
            if (modelIdInput.disabled && modelIdInput.value) {
                data.id = modelIdInput.value.trim();
            }
        }

        // 表单验证
        const requiredFields = ['id', 'name', 'target', 'type', 'url'];
        for (const field of requiredFields) {
            if (!data[field]) {
                this.showToast(`请填写${this.getFieldLabel(field)}`, 'error');
                return;
            }
        }

        // URL格式验证
        try {
            new URL(data.url);
        } catch {
            this.showToast('请输入有效的URL地址', 'error');
            return;
        }

        // 处理 prompt_value
        if (data.prompt_value && data.prompt_value.trim()) {
            try {
                if (data.prompt_value_type === 'object') {
                    data.prompt_value = JSON.parse(data.prompt_value);
                }
            } catch (error) {
                this.showToast('Prompt值JSON格式错误', 'error');
                return;
            }
        } else {
            // 如果prompt_value为空，设置为null以便后端清空该字段
            data.prompt_value = null;
        }

        // 添加更新时间
        data.updated_at = new Date().toISOString();

        // 显示加载状态
        const saveBtn = document.querySelector('#model-form button[type="submit"]');
        const originalText = saveBtn.innerHTML;
        saveBtn.innerHTML = '<i class="fas fa-spinner fa-spin mr-2"></i>保存中...';
        saveBtn.disabled = true;

        try {
            if (this.currentEditingModel) {
                // 更新模型
                await this.apiRequest(`/models/${this.currentEditingModel.id}`, {
                    method: 'PUT',
                    body: JSON.stringify(data)
                });
                this.showToast('✅ 模型更新成功', 'success');
            } else {
                // 创建模型
                await this.apiRequest('/models', {
                    method: 'POST',
                    body: JSON.stringify(data)
                });
                this.showToast('✅ 模型创建成功', 'success');
            }

            this.closeModal();
            this.loadModels();
        } catch (error) {
            this.showToast('❌ 操作失败: ' + error.message, 'error');
            console.error('保存模型失败:', error);
        } finally {
            // 恢复按钮状态
            saveBtn.innerHTML = originalText;
            saveBtn.disabled = false;
        }
    }

    getFieldLabel(field) {
        const labels = {
            'id': '模型ID',
            'name': '模型名称',
            'target': '目标模型',
            'type': '模型类型',
            'url': 'API地址'
        };
        return labels[field] || field;
    }

    async editModel(modelId) {
        const model = this.models.find(m => m.id === modelId);
        if (model) {
            this.openModal(model);
        }
    }

    deleteModel(modelId, modelName) {
        this.currentDeletingModel = modelId;
        document.getElementById('delete-model-name').textContent = modelName;
        document.getElementById('delete-modal').classList.remove('hidden');
        document.body.style.overflow = 'hidden';
    }

    closeDeleteModal() {
        document.getElementById('delete-modal').classList.add('hidden');
        document.body.style.overflow = 'auto';
        this.currentDeletingModel = null;
    }

    async confirmDelete() {
        if (!this.currentDeletingModel) return;

        try {
            await this.apiRequest(`/models/${this.currentDeletingModel}`, {
                method: 'DELETE'
            });
            this.showToast('✅ 模型删除成功', 'success');
            this.closeDeleteModal();
            this.loadModels();
        } catch (error) {
            console.error('删除模型失败:', error);
            this.showToast('❌ 删除失败: ' + error.message, 'error');
        }
    }

    async reloadConfig() {
        const button = document.getElementById('reload-config');
        const originalText = button.innerHTML;
        
        button.innerHTML = '<i class="fas fa-spinner fa-spin mr-2"></i>重新加载中...';
        button.disabled = true;

        try {
            await this.apiRequest('/config/reload', {
                method: 'POST'
            });
            this.showToast('✅ 配置重新加载成功', 'success');
            this.loadModels();
            this.loadStatus();
        } catch (error) {
            console.error('重新加载配置失败:', error);
            this.showToast('❌ 重新加载失败: ' + error.message, 'error');
        } finally {
            button.innerHTML = originalText;
            button.disabled = false;
        }
    }

    updateTotalCount() {
        document.getElementById('total-models').textContent = this.models.length;
    }

    updateModelCounts() {
        // 使用当前显示的模型列表进行统计
        const modelsToCount = this.filteredModels.length > 0 ? this.models : [];
        
        const chatCount = modelsToCount.filter(m => m.type === 'chat').length;
        const imageCount = modelsToCount.filter(m => m.type === 'image').length;
        const audioCount = modelsToCount.filter(m => m.type === 'audio').length;
        const videoCount = modelsToCount.filter(m => m.type === 'video').length;

        const chatElement = document.getElementById('chat-models-count');
        const imageElement = document.getElementById('image-models-count');
        const audioElement = document.getElementById('audio-models-count');
        const videoElement = document.getElementById('video-models-count');

        if (chatElement) chatElement.textContent = chatCount;
        if (imageElement) imageElement.textContent = imageCount;
        if (audioElement) audioElement.textContent = audioCount;
        if (videoElement) videoElement.textContent = videoCount;
        
        // 同时更新总数
        this.updateTotalCount();
    }

    showToast(message, type = 'info') {
        const container = document.getElementById('toast-container');
        const toast = document.createElement('div');
        
        const colors = {
            'success': 'bg-green-500',
            'error': 'bg-red-500',
            'warning': 'bg-yellow-500',
            'info': 'bg-blue-500'
        };

        const icons = {
            'success': 'fas fa-check-circle',
            'error': 'fas fa-exclamation-circle',
            'warning': 'fas fa-exclamation-triangle',
            'info': 'fas fa-info-circle'
        };

        toast.className = `${colors[type]} text-white px-6 py-3 rounded-lg shadow-lg flex items-center space-x-3 transform transition-all duration-300 translate-x-full`;
        toast.innerHTML = `
            <i class="${icons[type]}"></i>
            <span>${this.escapeHtml(message)}</span>
            <button onclick="this.parentElement.remove()" class="ml-4 text-white hover:text-gray-200">
                <i class="fas fa-times"></i>
            </button>
        `;

        container.appendChild(toast);

        // 动画显示
        setTimeout(() => {
            toast.classList.remove('translate-x-full');
        }, 100);

        // 自动消失
        setTimeout(() => {
            toast.classList.add('translate-x-full');
            setTimeout(() => {
                if (toast.parentElement) {
                    toast.remove();
                }
            }, 300);
        }, 5000);
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    async copyToClipboard(text, label = '内容') {
        try {
            await navigator.clipboard.writeText(text);
            this.showToast(`✅ ${label}已复制到剪贴板`, 'success');
        } catch (error) {
            // 降级方案：使用传统方法
            const textArea = document.createElement('textarea');
            textArea.value = text;
            document.body.appendChild(textArea);
            textArea.select();
            try {
                document.execCommand('copy');
                this.showToast(`✅ ${label}已复制到剪贴板`, 'success');
            } catch (fallbackError) {
                this.showToast(`❌ 复制失败`, 'error');
            }
            document.body.removeChild(textArea);
        }
    }

    formatDateTime(dateString) {
        if (!dateString) return '';
        try {
            const date = new Date(dateString);
            return date.toLocaleString('zh-CN', {
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit'
            });
        } catch (error) {
            return dateString;
        }
    }

    toggleAdvancedPromptConfig() {
        const content = document.getElementById('advanced-prompt-content');
        const icon = document.getElementById('advanced-prompt-icon');
        
        if (!content || !icon) return;
        
        if (content.classList.contains('hidden')) {
            // 展开
            content.classList.remove('hidden');
            content.classList.add('animate-slide-down');
            icon.classList.add('rotate-180');
        } else {
            // 收起
            content.classList.add('hidden');
            content.classList.remove('animate-slide-down');
            icon.classList.remove('rotate-180');
        }
    }
}

// 初始化应用
const app = new AIPromptProxyAdmin();

// 全局错误处理
window.addEventListener('unhandledrejection', (event) => {
    console.error('未处理的Promise拒绝:', event.reason);
    app.showToast('发生未知错误，请检查控制台', 'error');
});

// 键盘快捷键
document.addEventListener('keydown', (e) => {
    // Ctrl/Cmd + N 添加新模型
    if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
        e.preventDefault();
        app.openModal();
    }
    
    // Ctrl/Cmd + R 重新加载配置
    if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
        e.preventDefault();
        app.reloadConfig();
    }
    
    // Ctrl/Cmd + F 聚焦搜索框
    if ((e.ctrlKey || e.metaKey) && e.key === 'f') {
        e.preventDefault();
        const searchInput = document.getElementById('search-input');
        if (searchInput) {
            searchInput.focus();
            searchInput.select();
        }
    }
    
    // ESC 关闭模态框
    if (e.key === 'Escape') {
        const modelModal = document.getElementById('model-modal');
        const deleteModal = document.getElementById('delete-modal');
        
        if (!modelModal.classList.contains('hidden')) {
            app.closeModal();
        } else if (!deleteModal.classList.contains('hidden')) {
            app.closeDeleteModal();
        }
    }
});