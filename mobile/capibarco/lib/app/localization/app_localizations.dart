import 'package:flutter/material.dart';

class AppLocalizations {
  const AppLocalizations(this.locale);

  final Locale locale;

  static const supportedLocales = <Locale>[Locale('en'), Locale('ru')];

  static const delegate = _AppLocalizationsDelegate();

  static AppLocalizations of(BuildContext context) {
    final localization = Localizations.of<AppLocalizations>(
      context,
      AppLocalizations,
    );
    assert(localization != null, 'AppLocalizations not found in context');
    return localization!;
  }

  static const _localizedValues = <String, Map<String, String>>{
    'en': {
      'appTitle': 'Capibarco',
      'welcomeTitle': 'Find a warm home for every capy-friend',
      'welcomeSubtitle': 'Production-ready client for PetMatch microservices.',
      'signIn': 'Sign in',
      'createAccount': 'Create account',
      'email': 'Email',
      'password': 'Password',
      'confirmPassword': 'Confirm password',
      'noAccount': 'Need an account?',
      'haveAccount': 'Already have an account?',
      'feed': 'Feed',
      'discover': 'Discover',
      'notifications': 'Notifications',
      'profile': 'Profile',
      'retry': 'Retry',
      'save': 'Save changes',
      'signOut': 'Sign out',
      'editProfile': 'Edit profile',
      'city': 'City',
      'bio': 'Bio',
      'searchProfiles': 'Search profiles',
      'searchHint': 'Name, shelter, kennel...',
      'emptyFeed': 'No animals available right now.',
      'emptyProfiles': 'No matching profiles yet.',
      'emptyNotifications': 'No notifications yet.',
      'loading': 'Loading...',
      'pass': 'Pass',
      'like': 'Like',
      'staleData': 'Showing cached data because the network is unavailable.',
      'sessionExpired': 'Session expired. Please sign in again.',
      'profileUpdated': 'Profile updated.',
    },
    'ru': {
      'appTitle': 'Capibarco',
      'welcomeTitle': 'Тёплый дом для каждого капи-друга',
      'welcomeSubtitle': 'Production-ready клиент для микросервисов PetMatch.',
      'signIn': 'Войти',
      'createAccount': 'Создать аккаунт',
      'email': 'Email',
      'password': 'Пароль',
      'confirmPassword': 'Подтвердите пароль',
      'noAccount': 'Нет аккаунта?',
      'haveAccount': 'Уже есть аккаунт?',
      'feed': 'Лента',
      'discover': 'Поиск',
      'notifications': 'Уведомления',
      'profile': 'Профиль',
      'retry': 'Повторить',
      'save': 'Сохранить',
      'signOut': 'Выйти',
      'editProfile': 'Редактировать профиль',
      'city': 'Город',
      'bio': 'О себе',
      'searchProfiles': 'Поиск профилей',
      'searchHint': 'Имя, приют, питомник...',
      'emptyFeed': 'Сейчас доступных питомцев нет.',
      'emptyProfiles': 'Подходящие профили пока не найдены.',
      'emptyNotifications': 'Пока нет уведомлений.',
      'loading': 'Загрузка...',
      'pass': 'Пропустить',
      'like': 'Лайк',
      'staleData': 'Показаны кешированные данные, потому что сеть недоступна.',
      'sessionExpired': 'Сессия истекла. Войдите снова.',
      'profileUpdated': 'Профиль обновлён.',
    },
  };

  String _text(String key) =>
      _localizedValues[locale.languageCode]?[key] ??
      _localizedValues['en']![key]!;

  String get appTitle => _text('appTitle');
  String get welcomeTitle => _text('welcomeTitle');
  String get welcomeSubtitle => _text('welcomeSubtitle');
  String get signIn => _text('signIn');
  String get createAccount => _text('createAccount');
  String get email => _text('email');
  String get password => _text('password');
  String get confirmPassword => _text('confirmPassword');
  String get noAccount => _text('noAccount');
  String get haveAccount => _text('haveAccount');
  String get feed => _text('feed');
  String get discover => _text('discover');
  String get notifications => _text('notifications');
  String get profile => _text('profile');
  String get retry => _text('retry');
  String get save => _text('save');
  String get signOut => _text('signOut');
  String get editProfile => _text('editProfile');
  String get city => _text('city');
  String get bio => _text('bio');
  String get searchProfiles => _text('searchProfiles');
  String get searchHint => _text('searchHint');
  String get emptyFeed => _text('emptyFeed');
  String get emptyProfiles => _text('emptyProfiles');
  String get emptyNotifications => _text('emptyNotifications');
  String get loading => _text('loading');
  String get pass => _text('pass');
  String get like => _text('like');
  String get staleData => _text('staleData');
  String get sessionExpired => _text('sessionExpired');
  String get profileUpdated => _text('profileUpdated');
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  bool isSupported(Locale locale) => AppLocalizations.supportedLocales.any(
    (supported) => supported.languageCode == locale.languageCode,
  );

  @override
  Future<AppLocalizations> load(Locale locale) async =>
      AppLocalizations(locale);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}
