// AI Prompt Proxy 管理后台 JavaScript

class AIPromptProxyAdmin {
    constructor() {
        this.baseURL = 'http://localhost:8081/api/v1';
        this.models = [];
        this.filteredModels = [];
        this.currentEditingModel = null;
        this.token = localStorage.getItem('auth_token');
        this.isAuthenticated = false;
        this.publicKey = null;
        
        // 用户管理相关属性
        this.users = [];
        this.filteredUsers = [];
        this.currentEditingUser = null;
        this.currentView = 'models'; // 'models' 或 'users'
        
        this.init();
    }

    init() {
        this.checkAuthStatus();
    }

    async checkAuthStatus() {
        try {
            // 检查是否首次安装
            const installResponse = await fetch(`${this.baseURL}/auth/check-install`);
            const installData = await installResponse.json();
            
            if (installData.data.is_first_install) {
                this.showInstallPage();
                return;
            }
            
            // 检查是否已登录
            if (this.token) {
                try {
                    const response = await this.apiRequest('/auth/profile');
                    if (response.code === 0) {
                        this.isAuthenticated = true;
                        this.currentUser = response.data; // 存储当前用户信息
                        this.showMainApp();
                        this.initMainApp();
                        return;
                    }
                } catch (error) {
                    // Token无效，清除并显示登录页面
                    localStorage.removeItem('auth_token');
                    this.token = null;
                }
            }
            
            this.showLoginPage();
        } catch (error) {
            console.error('检查认证状态失败:', error);
            this.showLoginPage();
        }
    }

    showLoginPage() {
        document.getElementById('login-page').classList.remove('hidden');
        document.getElementById('install-page').classList.add('hidden');
        document.getElementById('main-app').classList.add('hidden');
        this.bindLoginEvents();
    }

    showInstallPage() {
        document.getElementById('login-page').classList.add('hidden');
        document.getElementById('install-page').classList.remove('hidden');
        document.getElementById('main-app').classList.add('hidden');
        this.bindInstallEvents();
    }

    showMainApp() {
        console.log('显示主应用页面');
        const loginPage = document.getElementById('login-page');
        const installPage = document.getElementById('install-page');
        const mainApp = document.getElementById('main-app');
        
        if (loginPage) {
            loginPage.classList.add('hidden');
            console.log('隐藏登录页面');
        }
        if (installPage) {
            installPage.classList.add('hidden');
            console.log('隐藏安装页面');
        }
        if (mainApp) {
            mainApp.classList.remove('hidden');
            console.log('显示主应用页面');
        } else {
            console.error('找不到主应用页面元素');
        }
    }

    initMainApp() {
        this.bindEvents();
        
        // 设置默认视图为模型管理
        this.currentView = 'models';
        this.showModelManagement();
        
        this.loadModels();
        this.loadStatus();
        
        // 根据用户权限控制界面显示
        this.setupUserInterface();
        
        // 定期刷新状态
        setInterval(() => this.loadStatus(), 30000);
        
        // 添加工具提示功能
        this.initTooltips();
    }
    
    setupUserInterface() {
        // 根据用户权限控制用户管理导航按钮的显示
        const userManagementNav = document.getElementById('user-management-nav');
        if (userManagementNav) {
            if (this.currentUser && this.currentUser.is_admin) {
                userManagementNav.style.display = 'flex';
            } else {
                userManagementNav.style.display = 'none';
            }
        }
        
        // 更新用户信息显示
        this.updateUserDisplay();
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

    bindLoginEvents() {
        const loginForm = document.getElementById('login-form');
        if (loginForm) {
            loginForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.handleLogin();
            });
        }
    }

    bindInstallEvents() {
        const installForm = document.getElementById('install-form');
        if (installForm) {
            installForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.handleInstall();
            });
        }
    }

    bindEvents() {
        console.log('开始绑定事件...');
        
        try {
            // 注销按钮
            const logoutBtn = document.getElementById('logout-btn');
            if (logoutBtn) {
                logoutBtn.addEventListener('click', () => {
                    this.handleLogout();
                });
            }

            // 添加模型按钮 - 支持多个按钮
            const addModelBtn = document.getElementById('add-model');
            if (addModelBtn) {
                addModelBtn.addEventListener('click', () => {
                    this.openModal();
                });
            } else {
                console.warn('未找到add-model按钮');
            }

            // 添加第一个模型按钮（空状态）
            const addFirstModelBtn = document.getElementById('add-first-model');
            if (addFirstModelBtn) {
                addFirstModelBtn.addEventListener('click', () => {
                    this.openModal();
                });
            }

            // 搜索和筛选功能
            const searchInput = document.getElementById('search-input');
            const typeFilter = document.getElementById('type-filter');
            const sortFilter = document.getElementById('sort-filter');

            if (searchInput) {
                searchInput.addEventListener('input', () => {
                    this.filterModels();
                });
            }

            if (typeFilter) {
                typeFilter.addEventListener('change', () => {
                    this.filterModels();
                });
            }

            if (sortFilter) {
                sortFilter.addEventListener('change', () => {
                    this.filterModels();
                });
            }

            // 导航栏模型管理按钮
            const modelManagementNav = document.getElementById('model-management-nav');
            if (modelManagementNav) {
                modelManagementNav.addEventListener('click', () => {
                    this.showModelManagement();
                });
            }

            // 导航栏添加模型按钮（保持兼容性）
            const addModelNav = document.getElementById('add-model-nav');
            if (addModelNav) {
                addModelNav.addEventListener('click', () => {
                    this.openModal();
                });
            }

            // 模态框关闭
            const closeModal = document.getElementById('close-modal');
            if (closeModal) {
                closeModal.addEventListener('click', () => {
                    this.closeModal();
                });
            }
            
            const cancelModal = document.getElementById('cancel-modal');
            if (cancelModal) {
                cancelModal.addEventListener('click', () => {
                    this.closeModal();
                });
            }

            // 删除模态框
            const cancelDelete = document.getElementById('cancel-delete');
            if (cancelDelete) {
                cancelDelete.addEventListener('click', () => {
                    this.closeDeleteModal();
                });
            }
            
            const confirmDelete = document.getElementById('confirm-delete');
            if (confirmDelete) {
                confirmDelete.addEventListener('click', () => {
                    this.confirmDelete();
                });
            }

            // 表单提交
            const modelForm = document.getElementById('model-form');
            if (modelForm) {
                modelForm.addEventListener('submit', (e) => {
                    e.preventDefault();
                    this.saveModel();
                });
            }





            // 模型类型变化时的处理
            const modelType = document.getElementById('model-type');
            if (modelType) {
                modelType.addEventListener('change', (e) => {
                    this.handleModelTypeChange(e.target.value);
                });
            }

            // 点击模态框外部关闭
            const modelModal = document.getElementById('model-modal');
            if (modelModal) {
                modelModal.addEventListener('click', (e) => {
                    if (e.target.id === 'model-modal') {
                        this.closeModal();
                    }
                });
            }
            
            const deleteModal = document.getElementById('delete-modal');
            if (deleteModal) {
                deleteModal.addEventListener('click', (e) => {
                    if (e.target.id === 'delete-modal') {
                        this.closeDeleteModal();
                    }
                });
            }

            // 高级配置抽屉展开/收起
            document.addEventListener('click', (e) => {
                if (e.target.id === 'toggle-advanced-prompt' || e.target.closest('#toggle-advanced-prompt')) {
                    this.toggleAdvancedPromptConfig();
                }
            });

            // 用户下拉菜单事件绑定
            this.bindUserDropdownEvents();

            // 用户管理相关事件绑定
            this.bindUserManagementEvents();
            
            console.log('事件绑定完成');
        } catch (error) {
            console.error('绑定事件时出错:', error);
        }
    }

    bindUserDropdownEvents() {
        try {
            // 用户菜单按钮
            const userMenuButton = document.getElementById('user-menu-button');
            const userDropdownMenu = document.getElementById('user-dropdown-menu');
            const dropdownArrow = document.getElementById('dropdown-arrow');

            if (userMenuButton && userDropdownMenu) {
                userMenuButton.addEventListener('click', (e) => {
                    e.stopPropagation();
                    this.toggleUserDropdown();
                });

                // 点击页面其他地方关闭下拉菜单
                document.addEventListener('click', (e) => {
                    if (!userMenuButton.contains(e.target) && !userDropdownMenu.contains(e.target)) {
                        this.closeUserDropdown();
                    }
                });
            }

            // 修改密码按钮
            const changePasswordBtn = document.getElementById('change-password-btn');
            if (changePasswordBtn) {
                changePasswordBtn.addEventListener('click', () => {
                    this.closeUserDropdown();
                    this.openChangePasswordModal();
                });
            }

            console.log('用户下拉菜单事件绑定完成');
        } catch (error) {
            console.error('绑定用户下拉菜单事件时出错:', error);
        }
    }

    toggleUserDropdown() {
        const userDropdownMenu = document.getElementById('user-dropdown-menu');
        const dropdownArrow = document.getElementById('dropdown-arrow');
        
        if (userDropdownMenu && dropdownArrow) {
            const isHidden = userDropdownMenu.classList.contains('hidden');
            
            if (isHidden) {
                userDropdownMenu.classList.remove('hidden');
                // 强制重绘以确保动画正常
                userDropdownMenu.offsetHeight;
                userDropdownMenu.classList.add('show');
                dropdownArrow.classList.add('rotate');
            } else {
                userDropdownMenu.classList.remove('show');
                dropdownArrow.classList.remove('rotate');
                // 等待动画完成后隐藏元素
                setTimeout(() => {
                    if (!userDropdownMenu.classList.contains('show')) {
                        userDropdownMenu.classList.add('hidden');
                    }
                }, 200);
            }
        }
    }

    closeUserDropdown() {
        const userDropdownMenu = document.getElementById('user-dropdown-menu');
        const dropdownArrow = document.getElementById('dropdown-arrow');
        
        if (userDropdownMenu && dropdownArrow) {
            userDropdownMenu.classList.remove('show');
            dropdownArrow.classList.remove('rotate');
            setTimeout(() => {
                if (!userDropdownMenu.classList.contains('show')) {
                    userDropdownMenu.classList.add('hidden');
                }
            }, 200);
        }
    }

    openChangePasswordModal() {
        // 清空表单
        const form = document.getElementById('change-password-form');
        if (form) {
            form.reset();
        }
        
        // 显示模态框
        document.getElementById('change-password-modal').classList.remove('hidden');
    }

    updateUserDisplay() {
        if (this.currentUser) {
            // 更新下拉菜单按钮中的用户信息
            const userDisplayName = document.getElementById('user-display-name');
            const userDisplayRole = document.getElementById('user-display-role');
            const dropdownUserName = document.getElementById('dropdown-user-name');
            const dropdownUserEmail = document.getElementById('dropdown-user-email');

            if (userDisplayName) {
                userDisplayName.textContent = this.currentUser.username || '用户';
            }
            if (userDisplayRole) {
                userDisplayRole.textContent = this.currentUser.is_admin ? 'Administrator' : 'User';
            }
            if (dropdownUserName) {
                dropdownUserName.textContent = this.currentUser.username || '用户';
            }
            if (dropdownUserEmail) {
                dropdownUserEmail.textContent = this.currentUser.email || this.currentUser.username + '@example.com';
            }
        }
    }

    bindUserManagementEvents() {
        try {
            // 用户管理导航按钮
            const userManagementNav = document.getElementById('user-management-nav');
            if (userManagementNav) {
                userManagementNav.addEventListener('click', () => {
                    this.showUserManagement();
                });
            }

            // 模型管理导航按钮（返回模型管理）
            const modelManagementNavBtn = document.getElementById('model-management-nav');
            if (modelManagementNavBtn) {
                modelManagementNavBtn.addEventListener('click', () => {
                    this.showModelManagement();
                });
            }



            // 添加用户按钮
            const addUserBtn = document.getElementById('add-user');
            if (addUserBtn) {
                addUserBtn.addEventListener('click', () => {
                    this.openUserModal();
                });
            }

            // 添加第一个用户按钮（空状态页面）
            const addFirstUserBtn = document.getElementById('add-first-user');
            if (addFirstUserBtn) {
                addFirstUserBtn.addEventListener('click', () => {
                    this.openUserModal();
                });
            }

            // 用户模态框关闭按钮
            const closeUserModalBtn = document.getElementById('close-user-modal');
            if (closeUserModalBtn) {
                closeUserModalBtn.addEventListener('click', () => {
                    this.closeUserModal();
                });
            }

            // 用户模态框取消按钮
            const cancelUserModalBtn = document.getElementById('cancel-user-modal');
            if (cancelUserModalBtn) {
                cancelUserModalBtn.addEventListener('click', () => {
                    this.closeUserModal();
                });
            }

            // 用户表单提交
            const userForm = document.getElementById('user-form');
            if (userForm) {
                userForm.addEventListener('submit', (e) => {
                    e.preventDefault();
                    this.saveUser();
                });
            }

            // 生成密码按钮
            const generatePasswordBtn = document.getElementById('generate-password');
            if (generatePasswordBtn) {
                generatePasswordBtn.addEventListener('click', () => {
                    this.generatePassword();
                });
            }

            // 复制密码按钮
            const copyPasswordBtn = document.getElementById('copy-password');
            if (copyPasswordBtn) {
                copyPasswordBtn.addEventListener('click', () => {
                    this.copyPassword();
                });
            }

            // 删除用户模态框关闭按钮
            const closeDeleteUserModalBtn = document.getElementById('close-delete-user-modal');
            if (closeDeleteUserModalBtn) {
                closeDeleteUserModalBtn.addEventListener('click', () => {
                    this.closeDeleteUserModal();
                });
            }

            // 删除用户模态框取消按钮
            const cancelDeleteUserBtn = document.getElementById('cancel-delete-user');
            if (cancelDeleteUserBtn) {
                cancelDeleteUserBtn.addEventListener('click', () => {
                    this.closeDeleteUserModal();
                });
            }

            // 确认删除用户按钮
            const confirmDeleteUserBtn = document.getElementById('confirm-delete-user');
            if (confirmDeleteUserBtn) {
                confirmDeleteUserBtn.addEventListener('click', () => {
                    this.confirmDeleteUser();
                });
            }

            // 修改密码模态框关闭按钮
            const closeChangePasswordModalBtn = document.getElementById('close-change-password-modal');
            if (closeChangePasswordModalBtn) {
                closeChangePasswordModalBtn.addEventListener('click', () => {
                    this.closeChangePasswordModal();
                });
            }

            // 修改密码模态框取消按钮
            const cancelChangePasswordBtn = document.getElementById('cancel-change-password');
            if (cancelChangePasswordBtn) {
                cancelChangePasswordBtn.addEventListener('click', () => {
                    this.closeChangePasswordModal();
                });
            }

            // 修改密码表单提交
            const changePasswordForm = document.getElementById('change-password-form');
            if (changePasswordForm) {
                changePasswordForm.addEventListener('submit', (e) => {
                    e.preventDefault();
                    this.savePassword();
                });
            }

            // 点击模态框外部关闭用户相关模态框
            document.addEventListener('click', (e) => {
                if (e.target.id === 'user-modal') {
                    this.closeUserModal();
                }
                if (e.target.id === 'delete-user-modal') {
                    this.closeDeleteUserModal();
                }
                if (e.target.id === 'change-password-modal') {
                    this.closeChangePasswordModal();
                }
            });

            console.log('用户管理事件绑定完成');
        } catch (error) {
            console.error('绑定用户管理事件时出错:', error);
        }
    }

    async apiRequest(endpoint, options = {}) {
        try {
            const headers = {
                'Content-Type': 'application/json',
                ...options.headers
            };

            // 添加认证头
            if (this.token) {
                headers['Authorization'] = `Bearer ${this.token}`;
                console.log(`API请求 ${endpoint} 使用token:`, this.token.substring(0, 20) + '...');
            } else {
                console.log(`API请求 ${endpoint} 没有token`);
            }

            const response = await fetch(`${this.baseURL}${endpoint}`, {
                headers,
                ...options
            });

            if (!response.ok) {
                if (response.status === 401) {
                    // 认证失败，清除token并跳转到登录页面
                    localStorage.removeItem('auth_token');
                    this.token = null;
                    this.isAuthenticated = false;
                    this.showLoginPage();
                    throw new Error('认证失败，请重新登录');
                }
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            console.error('API请求失败:', error);
            if (error.message !== '认证失败，请重新登录') {
                this.showToast(`请求失败: ${error.message}`, 'error');
            }
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
        const searchInput = document.getElementById('search-input');
        const typeFilter = document.getElementById('type-filter');
        const sortFilter = document.getElementById('sort-filter');

        const searchTerm = searchInput ? searchInput.value.toLowerCase() : '';
        const selectedType = typeFilter ? typeFilter.value : '';
        const sortBy = sortFilter ? sortFilter.value : 'updated_at';

        // 筛选模型
        let filteredModels = this.models.filter(model => {
            // 搜索匹配
            const matchesSearch = !searchTerm || 
                model.id.toLowerCase().includes(searchTerm) ||
                model.name.toLowerCase().includes(searchTerm) ||
                (model.prompt && model.prompt.toLowerCase().includes(searchTerm));

            // 类型匹配
            const matchesType = !selectedType || model.type === selectedType;

            return matchesSearch && matchesType;
        });

        // 排序
        filteredModels.sort((a, b) => {
            switch (sortBy) {
                case 'updated_at':
                    return new Date(b.updated_at || 0) - new Date(a.updated_at || 0);
                case 'type':
                    return a.type.localeCompare(b.type);
                case 'recent':
                    return b.id.localeCompare(a.id);
                case 'name':
                    return a.name.localeCompare(b.name);
                default:
                    return new Date(b.updated_at || 0) - new Date(a.updated_at || 0);
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
                            <span class="font-semibold text-gray-700">服务商接入地址:</span>
                            <p class="text-gray-900 break-all mt-1">${this.escapeHtml(model.url)}</p>
                        </div>
                    </div>
                    <div class="flex items-start text-sm text-gray-600">
                        <div class="w-8 h-8 bg-gray-100 rounded-lg flex items-center justify-center mr-3 mt-0.5">
                            <i class="fas fa-globe text-gray-500"></i>
                        </div>
                        <div class="flex-1">
                            <div class="flex items-center justify-between">
                                <span class="font-semibold text-gray-700">接入地址:</span>
                                <div class="relative">
                                    <button onclick="app.toggleAccessUrlDropdown('${model.id}')" class="text-xs text-blue-600 hover:text-blue-800 flex items-center">
                                        <i class="fas fa-list mr-1"></i>
                                        查看所有选项
                                        <i class="fas fa-chevron-down ml-1"></i>
                                    </button>
                                    <div id="access-url-dropdown-${model.id}" class="hidden absolute right-0 top-full mt-1 w-80 bg-white border border-gray-200 rounded-lg shadow-lg z-10">
                                        <div class="p-3">
                                            <p class="text-xs text-gray-600 mb-2">选择适合的接入地址:</p>
                                            ${this.generateAccessUrlExamples(model.url).map((url, index) => `
                                                <div class="flex items-center justify-between py-1 px-2 hover:bg-gray-50 rounded">
                                                    <span class="text-xs text-gray-800 break-all flex-1 mr-2">${url}</span>
                                                    <button onclick="app.copyToClipboard('${url}', '接入地址'); app.toggleAccessUrlDropdown('${model.id}')" class="text-gray-400 hover:text-blue-500">
                                                        <i class="fas fa-copy text-xs"></i>
                                                    </button>
                                                </div>
                                            `).join('')}
                                        </div>
                                    </div>
                                </div>
                            </div>
                            <div class="flex items-center mt-1">
                                <p class="text-gray-900 break-all flex-1">${this.generateAccessUrlExamples(model.url)[0]}</p>
                                <button onclick="app.copyToClipboard('${this.generateAccessUrlExamples(model.url)[0]}', '接入地址')" class="tooltip text-gray-400 hover:text-blue-500 transition-colors duration-200 ml-2" data-tooltip="复制接入地址">
                                    <i class="fas fa-copy text-xs"></i>
                                </button>
                            </div>
                            <p class="text-xs text-gray-500 mt-1">
                                <i class="fas fa-info-circle mr-1"></i>
                                默认显示localhost地址，点击上方按钮查看更多IP选项
                            </p>
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
            // 根据模型类型设置placeholder
            this.handleModelTypeChange(model.type);
        } else {
            this.currentEditingModel = null;
            title.textContent = '添加模型配置';
            form.reset();
            document.getElementById('model-id').disabled = false;
            
            // 只设置模型类型的默认值，其他字段保持空白
            document.getElementById('model-type').value = 'chat';
            // 根据默认模型类型设置placeholder
            this.handleModelTypeChange('chat');
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
        // 根据模型类型更新服务商接入地址的placeholder
        const urlInput = document.getElementById('model-url');
        if (urlInput) {
            const placeholders = {
                'chat': 'https://ark.cn-beijing.volces.com/api/v3/chat/completions',
                'image': 'https://ark.cn-beijing.volces.com/api/v3/images/generations',
                'audio': 'https://api.example.com/v1/audio/transcriptions',
                'video': 'https://api.example.com/v1/video/generations'
            };
            
            urlInput.placeholder = placeholders[modelType] || 'https://api.example.com';
        }
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

    generateAccessUrl(serviceProviderUrl) {
        try {
            // 获取当前浏览器的协议和端口
            const currentLocation = window.location;
            const protocol = currentLocation.protocol; // http: 或 https:
            const port = currentLocation.port; // 端口号
            
            // 解析服务商接入地址，提取路径
            const url = new URL(serviceProviderUrl);
            const path = url.pathname;
            
            // 构建端口部分
            const portPart = port ? `:${port}` : '';
            
            // 生成通用的接入地址模板，使用 {HOST} 作为占位符
            // 用户可以将 {HOST} 替换为实际的IP地址或域名
            return `${protocol}//{HOST}${portPart}${path}`;
        } catch (error) {
            console.error('生成接入地址失败:', error);
            return '无效的服务商地址';
        }
    }

    generateAccessUrlExamples(serviceProviderUrl) {
        try {
            const currentLocation = window.location;
            const protocol = currentLocation.protocol;
            const port = currentLocation.port;
            const url = new URL(serviceProviderUrl);
            const path = url.pathname;
            const portPart = port ? `:${port}` : '';
            
            // 生成常用的访问地址示例
            const examples = [
                `${protocol}//localhost${portPart}${path}`,
                `${protocol}//127.0.0.1${portPart}${path}`,
                `${protocol}//192.168.1.100${portPart}${path}`,
                `${protocol}//your-domain.com${portPart}${path}`
            ];
            
            return examples;
        } catch (error) {
            console.error('生成接入地址示例失败:', error);
            return [];
        }
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

    // 检查是否支持加密（安全上下文）
    isSecureContext() {
        return window.isSecureContext && window.crypto && window.crypto.subtle;
    }

    // RSA加密相关方法
    async getPublicKey() {
        try {
            const response = await fetch(`${this.baseURL}/auth/public-key`);
            const data = await response.json();
            
            if (data.code === 0) {
                this.publicKey = data.data.public_key;
                console.log('获取公钥成功');
                return this.publicKey;
            } else {
                throw new Error(data.message || '获取公钥失败');
            }
        } catch (error) {
            console.error('获取公钥失败:', error);
            throw error;
        }
    }

    async importPublicKey(pemKey) {
        try {
            // 检查是否支持加密
            if (!this.isSecureContext()) {
                throw new Error('当前环境不支持加密功能');
            }
            
            // 移除PEM头尾和换行符
            const pemHeader = "-----BEGIN PUBLIC KEY-----";
            const pemFooter = "-----END PUBLIC KEY-----";
            const pemContents = pemKey.replace(pemHeader, '').replace(pemFooter, '').replace(/\s/g, '');
            
            // Base64解码
            const binaryDerString = atob(pemContents);
            const binaryDer = new Uint8Array(binaryDerString.length);
            for (let i = 0; i < binaryDerString.length; i++) {
                binaryDer[i] = binaryDerString.charCodeAt(i);
            }
            
            // 导入公钥
            const publicKey = await window.crypto.subtle.importKey(
                'spki',
                binaryDer.buffer,
                {
                    name: 'RSA-OAEP',
                    hash: 'SHA-256'
                },
                false,
                ['encrypt']
            );
            
            return publicKey;
        } catch (error) {
            console.error('导入公钥失败:', error);
            throw error;
        }
    }

    async encryptPassword(password) {
        try {
            // 检查是否支持加密
            if (!this.isSecureContext()) {
                console.warn('当前环境不支持加密，将使用普通登录');
                return null; // 返回null表示不使用加密
            }
            
            // 如果没有公钥，先获取
            if (!this.publicKey) {
                await this.getPublicKey();
            }
            
            // 导入公钥
            const publicKey = await this.importPublicKey(this.publicKey);
            
            // 加密密码
            const encoder = new TextEncoder();
            const data = encoder.encode(password);
            const encrypted = await window.crypto.subtle.encrypt(
                {
                    name: 'RSA-OAEP'
                },
                publicKey,
                data
            );
            
            // 转换为Base64
            const encryptedArray = new Uint8Array(encrypted);
            const encryptedBase64 = btoa(String.fromCharCode.apply(null, encryptedArray));
            
            return encryptedBase64;
        } catch (error) {
            console.error('密码加密失败:', error);
            // 如果加密失败，返回null表示使用普通登录
            return null;
        }
    }

    // 认证相关方法
    async handleLogin() {
        const username = document.getElementById('username').value;
        const password = document.getElementById('password').value;

        if (!username || !password) {
            this.showToast('请输入用户名和密码', 'error');
            return;
        }

        try {
            let response, data;
            
            // 尝试加密登录
            console.log('正在尝试加密登录...');
            const encryptedPassword = await this.encryptPassword(password);
            
            if (encryptedPassword) {
                // 使用加密登录
                console.log('使用加密登录');
                response = await fetch(`${this.baseURL}/auth/encrypted-login`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ 
                        username, 
                        encrypted_password: encryptedPassword 
                    })
                });
            } else {
                // 降级到普通登录
                console.log('降级到普通登录');
                response = await fetch(`${this.baseURL}/auth/login`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ 
                        username, 
                        password 
                    })
                });
            }

            data = await response.json();
            console.log('登录响应:', data);

            if (data.code === 0) {
                this.token = data.data.token;
                localStorage.setItem('auth_token', this.token);
                this.isAuthenticated = true;
                console.log('Token设置成功:', this.token);
                this.showToast(data.message || '登录成功', 'success');
                
                // 延迟一下再切换页面，确保Toast显示和token设置完成
                setTimeout(async () => {
                    console.log('切换到主应用页面，当前token:', this.token);
                    this.showMainApp();
                    
                    // 获取用户信息后再初始化主应用
                    try {
                        const profileResponse = await this.apiRequest('/auth/profile');
                        if (profileResponse.code === 0) {
                            this.currentUser = profileResponse.data; // 设置当前用户信息
                            console.log('用户信息获取成功:', this.currentUser);
                        }
                    } catch (error) {
                        console.error('获取用户信息失败:', error);
                    }
                    
                    // 再延迟一下确保页面切换完成后再初始化
                    setTimeout(() => {
                        console.log('初始化主应用');
                        this.initMainApp();
                    }, 100);
                }, 500);
            } else {
                this.showToast(data.message || '登录失败', 'error');
            }
        } catch (error) {
            console.error('登录失败:', error);
            this.showToast('登录失败: ' + error.message, 'error');
        }
    }

    async handleInstall() {
        const username = document.getElementById('install-username').value;
        const password = document.getElementById('install-password').value;
        const confirmPassword = document.getElementById('install-confirm-password').value;

        if (!username || !password || !confirmPassword) {
            this.showToast('请填写所有字段', 'error');
            return;
        }

        if (password !== confirmPassword) {
            this.showToast('两次输入的密码不一致', 'error');
            return;
        }

        if (password.length < 6) {
            this.showToast('密码长度至少6位', 'error');
            return;
        }

        try {
            let response, data;
            
            // 尝试加密注册
            console.log('正在尝试加密注册...');
            const encryptedPassword = await this.encryptPassword(password);
            
            if (encryptedPassword) {
                // 使用加密注册
                console.log('使用加密注册');
                response = await fetch(`${this.baseURL}/auth/encrypted-register`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ 
                        username, 
                        encrypted_password: encryptedPassword 
                    })
                });
            } else {
                // 降级到普通注册
                console.log('降级到普通注册');
                response = await fetch(`${this.baseURL}/auth/register`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ 
                        username, 
                        password 
                    })
                });
            }

            data = await response.json();
            console.log('安装响应:', data);

            if (data.code === 0) {
                this.token = data.data.token;
                localStorage.setItem('auth_token', this.token);
                this.isAuthenticated = true;
                console.log('Token设置成功:', this.token);
                this.showToast(data.message || '安装完成，欢迎使用！', 'success');
                
                // 延迟一下再切换页面，确保Toast显示和token设置完成
                setTimeout(async () => {
                    console.log('切换到主应用页面，当前token:', this.token);
                    this.showMainApp();
                    
                    // 获取用户信息后再初始化主应用
                    try {
                        const profileResponse = await this.apiRequest('/auth/profile');
                        if (profileResponse.code === 0) {
                            this.currentUser = profileResponse.data; // 设置当前用户信息
                            console.log('用户信息获取成功:', this.currentUser);
                        }
                    } catch (error) {
                        console.error('获取用户信息失败:', error);
                    }
                    
                    // 再延迟一下确保页面切换完成后再初始化
                    setTimeout(() => {
                        console.log('初始化主应用');
                        this.initMainApp();
                    }, 100);
                }, 500);
            } else {
                this.showToast(data.message || '安装失败', 'error');
            }
        } catch (error) {
            console.error('安装失败:', error);
            this.showToast('安装失败: ' + error.message, 'error');
        }
    }

    async handleLogout() {
        try {
            await this.apiRequest('/auth/logout', {
                method: 'POST'
            });
        } catch (error) {
            console.error('注销请求失败:', error);
        } finally {
            // 无论请求是否成功，都清除本地状态
            localStorage.removeItem('auth_token');
            this.token = null;
            this.isAuthenticated = false;
            this.showToast('已注销登录', 'info');
            this.showLoginPage();
        }
    }

    // 用户管理相关方法
    showUserManagement() {
        // 检查权限：只有管理员才能访问用户管理
        if (!this.currentUser || !this.currentUser.is_admin) {
            this.showToast('❌ 权限不足，只有管理员才能访问用户管理', 'error');
            return;
        }
        
        // 设置当前视图
        this.currentView = 'users';
        
        // 隐藏模型管理页面
        const modelManagementSection = document.getElementById('model-management-section');
        if (modelManagementSection) {
            modelManagementSection.classList.add('hidden');
        }
        
        // 显示用户管理界面
        const userManagementSection = document.getElementById('user-management-section');
        if (userManagementSection) {
            userManagementSection.classList.remove('hidden');
        }
        
        // 更新导航按钮状态
        const modelNavBtn = document.getElementById('model-management-nav');
        const userNavBtn = document.getElementById('user-management-nav');
        
        if (modelNavBtn) {
            modelNavBtn.classList.remove('bg-blue-600', 'text-white');
            modelNavBtn.classList.add('bg-white/20', 'text-white/80');
        }
        
        if (userNavBtn) {
            userNavBtn.classList.remove('bg-white/20', 'text-white/80');
            userNavBtn.classList.add('bg-blue-600', 'text-white');
        }
        
        // 加载用户数据
        this.loadUsers();
    }

    showModelManagement() {
        // 设置当前视图
        this.currentView = 'models';
        
        // 隐藏用户管理界面
        const userManagementSection = document.getElementById('user-management-section');
        if (userManagementSection) {
            userManagementSection.classList.add('hidden');
        }
        
        // 显示模型管理页面
        const modelManagementSection = document.getElementById('model-management-section');
        if (modelManagementSection) {
            modelManagementSection.classList.remove('hidden');
        }
        
        // 更新导航按钮状态
        const modelNavBtn = document.getElementById('model-management-nav');
        const userNavBtn = document.getElementById('user-management-nav');
        
        if (modelNavBtn) {
            modelNavBtn.classList.remove('bg-white/20', 'text-white/80');
            modelNavBtn.classList.add('bg-blue-600', 'text-white');
        }
        
        if (userNavBtn) {
            userNavBtn.classList.remove('bg-blue-600', 'text-white');
            userNavBtn.classList.add('bg-white/20', 'text-white/80');
        }
        
        // 重新渲染模型列表
        this.filterModels();
    }

    async loadUsers() {
        try {
            const response = await this.apiRequest('/users');
            this.users = response.data.users || [];
            this.filteredUsers = [...this.users];
            this.updateUserCounts();
            this.renderUsers();
        } catch (error) {
            console.error('加载用户失败:', error);
            this.showToast('❌ 加载用户失败: ' + error.message, 'error');
        }
    }

    filterUsers() {
        // 移除搜索和筛选功能，直接显示所有用户
        this.filteredUsers = this.users;
        this.renderUsers();
    }

    renderUsers() {
        const container = document.getElementById('users-table-body');
        const emptyState = document.getElementById('users-empty-state');
        const userTable = container.closest('.bg-white\\/80'); // 用户表格容器

        if (this.filteredUsers.length === 0) {
            container.innerHTML = '';
            if (userTable) userTable.style.display = 'none'; // 隐藏用户表格
            emptyState.classList.remove('hidden');
            return;
        }

        emptyState.classList.add('hidden');
        if (userTable) userTable.style.display = 'block'; // 显示用户表格
        container.innerHTML = this.filteredUsers.map(user => `
            <tr class="hover:bg-gray-50 transition-colors duration-200">
                <td class="px-6 py-4 whitespace-nowrap">
                    <div class="flex items-center">
                        <div class="w-10 h-10 bg-gradient-to-br ${user.is_admin ? 'from-purple-400 to-purple-600' : 'from-blue-400 to-blue-600'} rounded-full flex items-center justify-center">
                            <i class="fas ${user.is_admin ? 'fa-crown' : 'fa-user'} text-white"></i>
                        </div>
                        <div class="ml-4">
                            <div class="text-sm font-medium text-gray-900">${this.escapeHtml(user.username)}</div>
                            <div class="text-sm text-gray-500">ID: ${user.id}</div>
                        </div>
                    </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${user.is_admin ? 'bg-purple-100 text-purple-800' : 'bg-blue-100 text-blue-800'}">
                        ${user.is_admin ? '👑 管理员' : '👤 普通用户'}
                    </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${user.is_enabled ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}">
                        ${user.is_enabled ? '✅ 启用' : '❌ 禁用'}
                    </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    ${user.last_login ? this.formatDateTime(user.last_login) : '从未登录'}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    ${this.formatDateTime(user.created_at)}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <div class="flex items-center justify-end space-x-2">
                        <button onclick="app.editUser(${user.id})" class="text-blue-600 hover:text-blue-900 transition-colors duration-200" title="编辑用户">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button onclick="app.toggleUserStatus(${user.id}, ${!user.is_enabled})" class="text-${user.is_enabled ? 'orange' : 'green'}-600 hover:text-${user.is_enabled ? 'orange' : 'green'}-900 transition-colors duration-200" title="${user.is_enabled ? '禁用用户' : '启用用户'}">
                            <i class="fas fa-${user.is_enabled ? 'ban' : 'check'}"></i>
                        </button>
                        <button onclick="app.changeUserPassword(${user.id}, '${this.escapeHtml(user.username)}')" class="text-purple-600 hover:text-purple-900 transition-colors duration-200" title="修改密码">
                            <i class="fas fa-key"></i>
                        </button>
                        <button onclick="app.deleteUser(${user.id}, '${this.escapeHtml(user.username)}')" class="text-red-600 hover:text-red-900 transition-colors duration-200" title="删除用户">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </td>
            </tr>
        `).join('');
    }

    updateUserCounts() {
        const totalUsers = this.users.length;
        const activeUsers = this.users.filter(u => u.is_enabled).length;

        const totalUsersEl = document.getElementById('total-users-count');
        const activeUsersEl = document.getElementById('active-users-count');
        
        if (totalUsersEl) totalUsersEl.textContent = totalUsers;
        if (activeUsersEl) activeUsersEl.textContent = activeUsers;
    }

    openUserModal(user = null) {
        const modal = document.getElementById('user-modal');
        const title = document.getElementById('user-modal-title');
        const form = document.getElementById('user-form');
        const passwordSection = document.getElementById('password-section');

        if (user) {
            this.currentEditingUser = user;
            title.textContent = '编辑用户';
            this.fillUserForm(user);
            passwordSection.style.display = 'none'; // 编辑时隐藏密码部分
        } else {
            this.currentEditingUser = null;
            title.textContent = '添加用户';
            form.reset();
            passwordSection.style.display = 'block'; // 新增时显示密码部分
            this.generatePassword(); // 自动生成密码
        }

        modal.classList.remove('hidden');
        document.body.style.overflow = 'hidden';
    }

    closeUserModal() {
        const modal = document.getElementById('user-modal');
        modal.classList.add('hidden');
        document.body.style.overflow = 'auto';
        this.currentEditingUser = null;
        
        // 清空生成的密码
        document.getElementById('generated-password').value = '';
        document.getElementById('copy-password').disabled = true;
    }

    fillUserForm(user) {
        document.getElementById('user-username').value = user.username;
        document.getElementById('user-is-admin').value = user.is_admin.toString();
    }

    generatePassword() {
        const length = 12;
        const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
        let password = '';
        
        for (let i = 0; i < length; i++) {
            password += charset.charAt(Math.floor(Math.random() * charset.length));
        }
        
        document.getElementById('generated-password').value = password;
        document.getElementById('copy-password').disabled = false;
        
        return password;
    }

    copyPassword() {
        const passwordInput = document.getElementById('generated-password');
        passwordInput.select();
        document.execCommand('copy');
        this.showToast('密码已复制到剪贴板', 'success');
    }

    async saveUser() {
        const form = document.getElementById('user-form');
        const formData = new FormData(form);
        const data = {
            username: formData.get('username').trim(),
            is_admin: formData.get('is_admin') === 'true'
        };

        // 创建用户时设置默认状态为启用
        if (!this.currentEditingUser) {
            data.is_enabled = true;
        }

        // 表单验证
        if (!data.username) {
            this.showToast('请输入用户名', 'error');
            return;
        }

        // 显示加载状态
        const saveBtn = document.querySelector('#user-form button[type="submit"]');
        const originalText = saveBtn.innerHTML;
        saveBtn.innerHTML = '<i class="fas fa-spinner fa-spin mr-2"></i>保存中...';
        saveBtn.disabled = true;

        try {
            if (this.currentEditingUser) {
                // 更新用户
                await this.apiRequest(`/users/${this.currentEditingUser.id}`, {
                    method: 'PUT',
                    body: JSON.stringify(data)
                });
                this.showToast('✅ 用户更新成功', 'success');
            } else {
                // 创建用户
                await this.apiRequest('/users', {
                    method: 'POST',
                    body: JSON.stringify(data)
                });
                this.showToast('✅ 用户创建成功', 'success');
            }

            this.closeUserModal();
            this.loadUsers();
        } catch (error) {
            this.showToast('❌ 操作失败: ' + error.message, 'error');
            console.error('保存用户失败:', error);
        } finally {
            // 恢复按钮状态
            saveBtn.innerHTML = originalText;
            saveBtn.disabled = false;
        }
    }

    async editUser(userId) {
        const user = this.users.find(u => u.id === userId);
        if (user) {
            this.openUserModal(user);
        }
    }

    deleteUser(userId, username) {
        this.currentDeletingUser = userId;
        document.getElementById('delete-user-name').textContent = username;
        document.getElementById('delete-user-modal').classList.remove('hidden');
        document.body.style.overflow = 'hidden';
    }

    closeDeleteUserModal() {
        document.getElementById('delete-user-modal').classList.add('hidden');
        document.body.style.overflow = 'auto';
        this.currentDeletingUser = null;
    }

    async confirmDeleteUser() {
        if (!this.currentDeletingUser) return;

        try {
            await this.apiRequest(`/users/${this.currentDeletingUser}`, {
                method: 'DELETE'
            });
            this.showToast('✅ 用户删除成功', 'success');
            this.closeDeleteUserModal();
            this.loadUsers();
        } catch (error) {
            console.error('删除用户失败:', error);
            this.showToast('❌ 删除失败: ' + error.message, 'error');
        }
    }

    async toggleUserStatus(userId, newStatus) {
        try {
            await this.apiRequest(`/users/${userId}/status`, {
                method: 'PUT',
                body: JSON.stringify({ is_enabled: newStatus })
            });
            this.showToast(`✅ 用户状态${newStatus ? '启用' : '禁用'}成功`, 'success');
            this.loadUsers();
        } catch (error) {
            console.error('更新用户状态失败:', error);
            this.showToast('❌ 状态更新失败: ' + error.message, 'error');
        }
    }

    changeUserPassword(userId, username) {
        this.currentPasswordUserId = userId;
        document.getElementById('change-password-subtitle').textContent = `修改用户 ${username} 的密码`;
        
        // 管理员修改其他用户密码时隐藏当前密码输入
        const currentPasswordSection = document.getElementById('current-password-section');
        const currentPasswordInput = document.getElementById('current-password');
        currentPasswordSection.style.display = 'none';
        currentPasswordInput.removeAttribute('required'); // 移除required属性避免表单验证错误
        
        document.getElementById('change-password-modal').classList.remove('hidden');
        document.body.style.overflow = 'hidden';
    }

    changeMyPassword() {
        this.currentPasswordUserId = null; // null表示修改自己的密码
        document.getElementById('change-password-subtitle').textContent = '修改我的密码';
        
        // 修改自己密码时显示当前密码输入
        const currentPasswordSection = document.getElementById('current-password-section');
        const currentPasswordInput = document.getElementById('current-password');
        currentPasswordSection.style.display = 'block';
        currentPasswordInput.setAttribute('required', 'required'); // 添加required属性
        
        document.getElementById('change-password-modal').classList.remove('hidden');
        document.body.style.overflow = 'hidden';
    }

    closeChangePasswordModal() {
        document.getElementById('change-password-modal').classList.add('hidden');
        document.body.style.overflow = 'auto';
        this.currentPasswordUserId = null;
        
        // 清空表单
        document.getElementById('change-password-form').reset();
    }

    async savePassword() {
        const form = document.getElementById('change-password-form');
        const formData = new FormData(form);
        const currentPassword = formData.get('current_password');
        const newPassword = formData.get('new_password');
        const confirmPassword = formData.get('confirm_password');

        // 表单验证
        if (!newPassword || !confirmPassword) {
            this.showToast('请填写新密码和确认密码', 'error');
            return;
        }

        if (newPassword !== confirmPassword) {
            this.showToast('两次输入的密码不一致', 'error');
            return;
        }

        if (newPassword.length < 6) {
            this.showToast('密码长度至少6位', 'error');
            return;
        }

        // 显示加载状态
        const saveBtn = document.querySelector('#change-password-form button[type="submit"]');
        const originalText = saveBtn.innerHTML;
        saveBtn.innerHTML = '<i class="fas fa-spinner fa-spin mr-2"></i>修改中...';
        saveBtn.disabled = true;

        try {
            if (this.currentPasswordUserId) {
                // 管理员修改其他用户密码
                await this.apiRequest(`/users/${this.currentPasswordUserId}/password`, {
                    method: 'PUT',
                    body: JSON.stringify({ new_password: newPassword })
                });
            } else {
                // 用户修改自己的密码
                if (!currentPassword) {
                    this.showToast('请输入当前密码', 'error');
                    return;
                }
                await this.apiRequest('/user/password', {
                    method: 'PUT',
                    body: JSON.stringify({ 
                        old_password: currentPassword,
                        new_password: newPassword 
                    })
                });
            }

            this.showToast('✅ 密码修改成功', 'success');
            this.closeChangePasswordModal();
        } catch (error) {
            this.showToast('❌ 密码修改失败: ' + error.message, 'error');
            console.error('修改密码失败:', error);
        } finally {
            // 恢复按钮状态
            saveBtn.innerHTML = originalText;
            saveBtn.disabled = false;
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