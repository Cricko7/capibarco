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
      'createKennelProfile': 'Kennel profile created.',
      'profileType': 'Profile type',
      'userProfile': 'User',
      'shelterProfile': 'Shelter',
      'kennelProfile': 'Kennel',
      'publishPet': 'Publish pet',
      'petName': 'Pet name',
      'species': 'Species',
      'breed': 'Breed',
      'sex': 'Sex',
      'size': 'Size',
      'ageMonths': 'Age in months',
      'vaccinated': 'Vaccinated',
      'sterilized': 'Sterilized',
      'publishNow': 'Publish immediately',
      'animalCreated': 'Pet profile created.',
      'saveDraft': 'Save draft',
      'draftSaved': 'Pet draft saved.',
      'petPublished': 'Pet profile published.',
      'photoRequiredForPublish': 'Add at least one photo before publishing.',
      'createPetCta': 'Add pet listing',
      'createProfileFirst': 'Create or update your profile first.',
      'petPhoto': 'Pet photo',
      'addPhoto': 'Add photo',
      'changePhoto': 'Change photo',
      'supportAnimal': 'Support this animal',
      'donationAmount': 'Donation amount',
      'amountHint': 'Enter amount in RUB',
      'donateNow': 'Create donation',
      'donationIntentCreated': 'Donation intent is ready.',
      'paymentUrl': 'Payment URL',
      'clientSecret': 'Client secret',
      'close': 'Close',
      'chat': 'Chat',
      'startChat': 'Start chat',
      'comments': 'Comments',
      'noComments': 'No comments yet.',
      'profilePets': 'Pet cards',
      'noProfilePets': 'This profile has no published pets yet.',
      'openProfile': 'Open profile',
      'messageHint': 'Write a message...',
      'send': 'Send',
      'chatReady': 'Chat is ready.',
      'chatUnavailable': 'Unable to open chat right now.',
      'publicProfileSubtitle': 'Public profile, reviews, and active pet cards.',
      'reviewsLabel': 'reviews',
      'aboutProfile': 'About profile',
      'profileActions': 'Actions',
      'publishedPets': 'Published pets',
    },
    'ru': {
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
      'createKennelProfile': 'Kennel profile created.',
      'profileType': 'Profile type',
      'userProfile': 'User',
      'shelterProfile': 'Shelter',
      'kennelProfile': 'Kennel',
      'publishPet': 'Publish pet',
      'petName': 'Pet name',
      'species': 'Species',
      'breed': 'Breed',
      'sex': 'Sex',
      'size': 'Size',
      'ageMonths': 'Age in months',
      'vaccinated': 'Vaccinated',
      'sterilized': 'Sterilized',
      'publishNow': 'Publish immediately',
      'animalCreated': 'Pet profile created.',
      'saveDraft': 'Save draft',
      'draftSaved': 'Pet draft saved.',
      'petPublished': 'Pet profile published.',
      'photoRequiredForPublish': 'Add at least one photo before publishing.',
      'createPetCta': 'Add pet listing',
      'createProfileFirst': 'Create or update your profile first.',
      'petPhoto': 'Pet photo',
      'addPhoto': 'Add photo',
      'changePhoto': 'Change photo',
      'supportAnimal': 'Support this animal',
      'donationAmount': 'Donation amount',
      'amountHint': 'Enter amount in RUB',
      'donateNow': 'Create donation',
      'donationIntentCreated': 'Donation intent is ready.',
      'paymentUrl': 'Payment URL',
      'clientSecret': 'Client secret',
      'close': 'Close',
      'chat': 'Чат',
      'startChat': 'Начать чат',
      'comments': 'Комментарии',
      'noComments': 'Комментариев пока нет.',
      'profilePets': 'Карточки питомцев',
      'noProfilePets': 'У этого профиля пока нет опубликованных питомцев.',
      'openProfile': 'Открыть профиль',
      'messageHint': 'Напишите сообщение...',
      'send': 'Отправить',
      'chatReady': 'Чат готов.',
      'chatUnavailable': 'Не удалось открыть чат.',
      'publicProfileSubtitle': 'Публичный профиль, отзывы и карточки питомцев.',
      'reviewsLabel': 'отзывов',
      'aboutProfile': 'О профиле',
      'profileActions': 'Действия',
      'publishedPets': 'Опубликованные питомцы',
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
  String get createKennelProfile => _text('createKennelProfile');
  String get profileType => _text('profileType');
  String get userProfile => _text('userProfile');
  String get shelterProfile => _text('shelterProfile');
  String get kennelProfile => _text('kennelProfile');
  String get publishPet => _text('publishPet');
  String get petName => _text('petName');
  String get species => _text('species');
  String get breed => _text('breed');
  String get sex => _text('sex');
  String get size => _text('size');
  String get ageMonths => _text('ageMonths');
  String get vaccinated => _text('vaccinated');
  String get sterilized => _text('sterilized');
  String get publishNow => _text('publishNow');
  String get animalCreated => _text('animalCreated');
  String get saveDraft => _text('saveDraft');
  String get draftSaved => _text('draftSaved');
  String get petPublished => _text('petPublished');
  String get photoRequiredForPublish => _text('photoRequiredForPublish');
  String get createPetCta => _text('createPetCta');
  String get createProfileFirst => _text('createProfileFirst');
  String get petPhoto => _text('petPhoto');
  String get addPhoto => _text('addPhoto');
  String get changePhoto => _text('changePhoto');
  String get supportAnimal => _text('supportAnimal');
  String get donationAmount => _text('donationAmount');
  String get amountHint => _text('amountHint');
  String get donateNow => _text('donateNow');
  String get donationIntentCreated => _text('donationIntentCreated');
  String get paymentUrl => _text('paymentUrl');
  String get clientSecret => _text('clientSecret');
  String get close => _text('close');
  String get chat => _text('chat');
  String get startChat => _text('startChat');
  String get comments => _text('comments');
  String get noComments => _text('noComments');
  String get profilePets => _text('profilePets');
  String get noProfilePets => _text('noProfilePets');
  String get openProfile => _text('openProfile');
  String get messageHint => _text('messageHint');
  String get send => _text('send');
  String get chatReady => _text('chatReady');
  String get chatUnavailable => _text('chatUnavailable');
  String get publicProfileSubtitle => _text('publicProfileSubtitle');
  String get reviewsLabel => _text('reviewsLabel');
  String get aboutProfile => _text('aboutProfile');
  String get profileActions => _text('profileActions');
  String get publishedPets => _text('publishedPets');
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
