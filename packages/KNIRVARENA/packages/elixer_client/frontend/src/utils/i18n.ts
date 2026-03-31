/**
 * Internationalization (i18n) Support
 * Provides multi-language support for KNIRV Controller
 */

import { useState, useEffect, useCallback, createContext, useContext } from 'react';

export type SupportedLanguage = 'en' | 'es' | 'fr' | 'de' | 'ja' | 'ko' | 'zh' | 'pt' | 'ru' | 'ar';

export interface LanguageInfo {
  code: SupportedLanguage;
  name: string;
  nativeName: string;
  flag: string;
  rtl?: boolean;
}

export const supportedLanguages: LanguageInfo[] = [
  { code: 'en', name: 'English', nativeName: 'English', flag: '🇺🇸' },
  { code: 'es', name: 'Spanish', nativeName: 'Español', flag: '🇪🇸' },
  { code: 'fr', name: 'French', nativeName: 'Français', flag: '🇫🇷' },
  { code: 'de', name: 'German', nativeName: 'Deutsch', flag: '🇩🇪' },
  { code: 'ja', name: 'Japanese', nativeName: '日本語', flag: '🇯🇵' },
  { code: 'ko', name: 'Korean', nativeName: '한국어', flag: '🇰🇷' },
  { code: 'zh', name: 'Chinese', nativeName: '中文', flag: '🇨🇳' },
  { code: 'pt', name: 'Portuguese', nativeName: 'Português', flag: '🇵🇹' },
  { code: 'ru', name: 'Russian', nativeName: 'Русский', flag: '🇷🇺' },
  { code: 'ar', name: 'Arabic', nativeName: 'العربية', flag: '🇸🇦', rtl: true }
];

// Translation keys and default English translations
export const translations: Record<SupportedLanguage, Record<string, string>> = {
  en: {
    // Navigation
    'nav.home': 'Home',
    'nav.wallet': 'Wallet',
    'nav.skills': 'Skills',
    'nav.settings': 'Settings',
    'nav.help': 'Help',
    
    // Common actions
    'action.save': 'Save',
    'action.cancel': 'Cancel',
    'action.delete': 'Delete',
    'action.edit': 'Edit',
    'action.create': 'Create',
    'action.connect': 'Connect',
    'action.disconnect': 'Disconnect',
    'action.retry': 'Retry',
    'action.refresh': 'Refresh',
    'action.close': 'Close',
    'action.back': 'Back',
    'action.next': 'Next',
    'action.previous': 'Previous',
    'action.confirm': 'Confirm',
    
    // Status messages
    'status.loading': 'Loading...',
    'status.saving': 'Saving...',
    'status.connecting': 'Connecting...',
    'status.connected': 'Connected',
    'status.disconnected': 'Disconnected',
    'status.error': 'Error',
    'status.success': 'Success',
    'status.complete': 'Complete',
    
    // KNIRVANA Game
    'game.title': 'KNIRVANA Graph',
    'game.start': 'Start Game',
    'game.pause': 'Pause Game',
    'game.nrv.balance': 'NRV Balance',
    'game.deploy.agent': 'Deploy Agent',
    'game.create.agent': 'Create Agent',
    'game.merge.collective': 'Merge to Collective',
    'game.errors.resolved': 'Errors Resolved',
    'game.skills.learned': 'Skills Learned',
    
    // Wallet
    'wallet.title': 'XION Wallet',
    'wallet.balance': 'Balance',
    'wallet.address': 'Address',
    'wallet.connect': 'Connect Wallet',
    'wallet.disconnect': 'Disconnect Wallet',
    'wallet.transaction.history': 'Transaction History',
    'wallet.send': 'Send',
    'wallet.receive': 'Receive',
    
    // Skills
    'skills.title': 'AI Skills',
    'skills.proficiency': 'Proficiency',
    'skills.category': 'Category',
    'skills.add': 'Add Skill',
    'skills.edit': 'Edit Skill',
    'skills.delete': 'Delete Skill',
    
    // Settings
    'settings.title': 'Settings',
    'settings.language': 'Language',
    'settings.theme': 'Theme',
    'settings.notifications': 'Notifications',
    'settings.privacy': 'Privacy',
    'settings.security': 'Security',
    
    // Error messages
    'error.network': 'Network connection error',
    'error.wallet': 'Wallet connection failed',
    'error.invalid.input': 'Invalid input',
    'error.permission.denied': 'Permission denied',
    'error.not.found': 'Not found',
    'error.server': 'Server error',
    'error.unknown': 'Unknown error occurred',
    
    // Accessibility
    'a11y.menu.open': 'Open menu',
    'a11y.menu.close': 'Close menu',
    'a11y.modal.close': 'Close modal',
    'a11y.loading': 'Loading content',
    'a11y.error': 'Error message',
    'a11y.success': 'Success message'
  },
  
  es: {
    // Navigation
    'nav.home': 'Inicio',
    'nav.wallet': 'Billetera',
    'nav.skills': 'Habilidades',
    'nav.settings': 'Configuración',
    'nav.help': 'Ayuda',
    
    // Common actions
    'action.save': 'Guardar',
    'action.cancel': 'Cancelar',
    'action.delete': 'Eliminar',
    'action.edit': 'Editar',
    'action.create': 'Crear',
    'action.connect': 'Conectar',
    'action.disconnect': 'Desconectar',
    'action.retry': 'Reintentar',
    'action.refresh': 'Actualizar',
    'action.close': 'Cerrar',
    'action.back': 'Atrás',
    'action.next': 'Siguiente',
    'action.previous': 'Anterior',
    'action.confirm': 'Confirmar',
    
    // Status messages
    'status.loading': 'Cargando...',
    'status.saving': 'Guardando...',
    'status.connecting': 'Conectando...',
    'status.connected': 'Conectado',
    'status.disconnected': 'Desconectado',
    'status.error': 'Error',
    'status.success': 'Éxito',
    'status.complete': 'Completo',
    
    // KNIRVANA Game
    'game.title': 'Gráfico KNIRVANA',
    'game.start': 'Iniciar Juego',
    'game.pause': 'Pausar Juego',
    'game.nrv.balance': 'Balance NRV',
    'game.deploy.agent': 'Desplegar Agente',
    'game.create.agent': 'Crear Agente',
    'game.merge.collective': 'Fusionar al Colectivo',
    'game.errors.resolved': 'Errores Resueltos',
    'game.skills.learned': 'Habilidades Aprendidas'
  },
  
  fr: {
    // Navigation
    'nav.home': 'Accueil',
    'nav.wallet': 'Portefeuille',
    'nav.skills': 'Compétences',
    'nav.settings': 'Paramètres',
    'nav.help': 'Aide',
    
    // Common actions
    'action.save': 'Enregistrer',
    'action.cancel': 'Annuler',
    'action.delete': 'Supprimer',
    'action.edit': 'Modifier',
    'action.create': 'Créer',
    'action.connect': 'Connecter',
    'action.disconnect': 'Déconnecter',
    'action.retry': 'Réessayer',
    'action.refresh': 'Actualiser',
    'action.close': 'Fermer',
    'action.back': 'Retour',
    'action.next': 'Suivant',
    'action.previous': 'Précédent',
    'action.confirm': 'Confirmer'
  },
  
  de: {
    // Navigation
    'nav.home': 'Startseite',
    'nav.wallet': 'Geldbörse',
    'nav.skills': 'Fähigkeiten',
    'nav.settings': 'Einstellungen',
    'nav.help': 'Hilfe',
    
    // Common actions
    'action.save': 'Speichern',
    'action.cancel': 'Abbrechen',
    'action.delete': 'Löschen',
    'action.edit': 'Bearbeiten',
    'action.create': 'Erstellen',
    'action.connect': 'Verbinden',
    'action.disconnect': 'Trennen',
    'action.retry': 'Wiederholen',
    'action.refresh': 'Aktualisieren',
    'action.close': 'Schließen',
    'action.back': 'Zurück',
    'action.next': 'Weiter',
    'action.previous': 'Vorherige',
    'action.confirm': 'Bestätigen'
  },
  
  ja: {
    // Navigation
    'nav.home': 'ホーム',
    'nav.wallet': 'ウォレット',
    'nav.skills': 'スキル',
    'nav.settings': '設定',
    'nav.help': 'ヘルプ',
    
    // Common actions
    'action.save': '保存',
    'action.cancel': 'キャンセル',
    'action.delete': '削除',
    'action.edit': '編集',
    'action.create': '作成',
    'action.connect': '接続',
    'action.disconnect': '切断',
    'action.retry': '再試行',
    'action.refresh': '更新',
    'action.close': '閉じる',
    'action.back': '戻る',
    'action.next': '次へ',
    'action.previous': '前へ',
    'action.confirm': '確認'
  },
  
  ko: {
    // Navigation
    'nav.home': '홈',
    'nav.wallet': '지갑',
    'nav.skills': '스킬',
    'nav.settings': '설정',
    'nav.help': '도움말',
    
    // Common actions
    'action.save': '저장',
    'action.cancel': '취소',
    'action.delete': '삭제',
    'action.edit': '편집',
    'action.create': '생성',
    'action.connect': '연결',
    'action.disconnect': '연결 해제',
    'action.retry': '다시 시도',
    'action.refresh': '새로 고침',
    'action.close': '닫기',
    'action.back': '뒤로',
    'action.next': '다음',
    'action.previous': '이전',
    'action.confirm': '확인'
  },
  
  zh: {
    // Navigation
    'nav.home': '首页',
    'nav.wallet': '钱包',
    'nav.skills': '技能',
    'nav.settings': '设置',
    'nav.help': '帮助',
    
    // Common actions
    'action.save': '保存',
    'action.cancel': '取消',
    'action.delete': '删除',
    'action.edit': '编辑',
    'action.create': '创建',
    'action.connect': '连接',
    'action.disconnect': '断开连接',
    'action.retry': '重试',
    'action.refresh': '刷新',
    'action.close': '关闭',
    'action.back': '返回',
    'action.next': '下一个',
    'action.previous': '上一个',
    'action.confirm': '确认'
  },
  
  pt: {
    // Navigation
    'nav.home': 'Início',
    'nav.wallet': 'Carteira',
    'nav.skills': 'Habilidades',
    'nav.settings': 'Configurações',
    'nav.help': 'Ajuda',
    
    // Common actions
    'action.save': 'Salvar',
    'action.cancel': 'Cancelar',
    'action.delete': 'Excluir',
    'action.edit': 'Editar',
    'action.create': 'Criar',
    'action.connect': 'Conectar',
    'action.disconnect': 'Desconectar',
    'action.retry': 'Tentar novamente',
    'action.refresh': 'Atualizar',
    'action.close': 'Fechar',
    'action.back': 'Voltar',
    'action.next': 'Próximo',
    'action.previous': 'Anterior',
    'action.confirm': 'Confirmar'
  },
  
  ru: {
    // Navigation
    'nav.home': 'Главная',
    'nav.wallet': 'Кошелёк',
    'nav.skills': 'Навыки',
    'nav.settings': 'Настройки',
    'nav.help': 'Помощь',
    
    // Common actions
    'action.save': 'Сохранить',
    'action.cancel': 'Отмена',
    'action.delete': 'Удалить',
    'action.edit': 'Редактировать',
    'action.create': 'Создать',
    'action.connect': 'Подключить',
    'action.disconnect': 'Отключить',
    'action.retry': 'Повторить',
    'action.refresh': 'Обновить',
    'action.close': 'Закрыть',
    'action.back': 'Назад',
    'action.next': 'Далее',
    'action.previous': 'Предыдущий',
    'action.confirm': 'Подтвердить'
  },
  
  ar: {
    // Navigation
    'nav.home': 'الرئيسية',
    'nav.wallet': 'المحفظة',
    'nav.skills': 'المهارات',
    'nav.settings': 'الإعدادات',
    'nav.help': 'المساعدة',
    
    // Common actions
    'action.save': 'حفظ',
    'action.cancel': 'إلغاء',
    'action.delete': 'حذف',
    'action.edit': 'تحرير',
    'action.create': 'إنشاء',
    'action.connect': 'اتصال',
    'action.disconnect': 'قطع الاتصال',
    'action.retry': 'إعادة المحاولة',
    'action.refresh': 'تحديث',
    'action.close': 'إغلاق',
    'action.back': 'رجوع',
    'action.next': 'التالي',
    'action.previous': 'السابق',
    'action.confirm': 'تأكيد'
  }
};

// I18n Context
export interface I18nContextType {
  language: SupportedLanguage;
  setLanguage: (language: SupportedLanguage) => void;
  t: (key: string, fallback?: string) => string;
  isRTL: boolean;
  supportedLanguages: LanguageInfo[];
}

export const I18nContext = createContext<I18nContextType | null>(null);

// Translation function
export const createTranslationFunction = (language: SupportedLanguage) => {
  return (key: string, fallback?: string): string => {
    const translation = translations[language]?.[key] || translations.en[key] || fallback || key;
    return translation;
  };
};

// Detect browser language
export const detectBrowserLanguage = (): SupportedLanguage => {
  const browserLang = navigator.language.split('-')[0] as SupportedLanguage;
  return supportedLanguages.find(lang => lang.code === browserLang)?.code || 'en';
};

// Get stored language preference
export const getStoredLanguage = (): SupportedLanguage => {
  try {
    const stored = localStorage.getItem('knirv_language') as SupportedLanguage;
    return supportedLanguages.find(lang => lang.code === stored)?.code || detectBrowserLanguage();
  } catch {
    return detectBrowserLanguage();
  }
};

// Store language preference
export const storeLanguage = (language: SupportedLanguage): void => {
  try {
    localStorage.setItem('knirv_language', language);
  } catch (error) {
    console.warn('Failed to store language preference:', error);
  }
};

// Custom hook for i18n
export const useI18n = () => {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error('useI18n must be used within an I18nProvider');
  }
  return context;
};

// Format numbers according to locale
export const formatNumber = (number: number, language: SupportedLanguage): string => {
  try {
    return new Intl.NumberFormat(language === 'zh' ? 'zh-CN' : language).format(number);
  } catch {
    return number.toString();
  }
};

// Format currency according to locale
export const formatCurrency = (amount: number, currency: string, language: SupportedLanguage): string => {
  try {
    return new Intl.NumberFormat(language === 'zh' ? 'zh-CN' : language, {
      style: 'currency',
      currency: currency.toUpperCase()
    }).format(amount);
  } catch {
    return `${amount} ${currency}`;
  }
};

// Format date according to locale
export const formatDate = (date: Date, language: SupportedLanguage): string => {
  try {
    return new Intl.DateTimeFormat(language === 'zh' ? 'zh-CN' : language).format(date);
  } catch {
    return date.toLocaleDateString();
  }
};

// Format relative time according to locale
export const formatRelativeTime = (date: Date, language: SupportedLanguage): string => {
  try {
    const rtf = new Intl.RelativeTimeFormat(language === 'zh' ? 'zh-CN' : language, { numeric: 'auto' });
    const diffInSeconds = (date.getTime() - Date.now()) / 1000;

    if (Math.abs(diffInSeconds) < 60) {
      return rtf.format(Math.round(diffInSeconds), 'second');
    } else if (Math.abs(diffInSeconds) < 3600) {
      return rtf.format(Math.round(diffInSeconds / 60), 'minute');
    } else if (Math.abs(diffInSeconds) < 86400) {
      return rtf.format(Math.round(diffInSeconds / 3600), 'hour');
    } else {
      return rtf.format(Math.round(diffInSeconds / 86400), 'day');
    }
  } catch {
    return formatDate(date, language);
  }
};

// I18n Provider Hook
export const useI18nProvider = () => {
  const [language, setLanguageState] = useState<SupportedLanguage>(getStoredLanguage);

  const setLanguage = useCallback((newLanguage: SupportedLanguage) => {
    setLanguageState(newLanguage);
    storeLanguage(newLanguage);

    // Update document direction for RTL languages
    const languageInfo = supportedLanguages.find(lang => lang.code === newLanguage);
    document.documentElement.dir = languageInfo?.rtl ? 'rtl' : 'ltr';
    document.documentElement.lang = newLanguage;
  }, []);

  const t = useCallback((key: string, fallback?: string): string => {
    const translation = translations[language]?.[key] || translations.en[key] || fallback || key;
    return translation;
  }, [language]);

  const isRTL = supportedLanguages.find(lang => lang.code === language)?.rtl || false;

  useEffect(() => {
    // Set initial document direction and language
    const languageInfo = supportedLanguages.find(lang => lang.code === language);
    document.documentElement.dir = languageInfo?.rtl ? 'rtl' : 'ltr';
    document.documentElement.lang = language;
  }, [language]);

  return {
    language,
    setLanguage,
    t,
    isRTL,
    supportedLanguages
  };
};
