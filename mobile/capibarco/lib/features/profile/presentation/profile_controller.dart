import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../bootstrap/providers.dart';
import '../../../core/config/environment.dart';
import '../../../core/error/app_exception.dart';
import '../../../core/network/network_providers.dart';
import '../../../core/network/rest_service_client.dart';
import '../../auth/presentation/auth_controller.dart';
import '../domain/entities/profile_animal_card.dart';
import '../data/api/profile_api_client.dart';
import '../data/datasources/profile_remote_data_source.dart';
import '../data/repositories/profile_repository_impl.dart';
import '../domain/entities/user_profile.dart';

class ProfileState {
  const ProfileState({
    this.profile,
    this.animals = const <ProfileAnimalCardEntity>[],
    this.isLoading = false,
    this.isSaving = false,
    this.errorMessage,
    this.infoMessage,
  });

  final UserProfileEntity? profile;
  final List<ProfileAnimalCardEntity> animals;
  final bool isLoading;
  final bool isSaving;
  final String? errorMessage;
  final String? infoMessage;

  ProfileState copyWith({
    UserProfileEntity? profile,
    List<ProfileAnimalCardEntity>? animals,
    bool clearProfile = false,
    bool? isLoading,
    bool? isSaving,
    String? errorMessage,
    bool clearError = false,
    String? infoMessage,
    bool clearInfo = false,
  }) {
    return ProfileState(
      profile: clearProfile ? null : (profile ?? this.profile),
      animals: animals ?? this.animals,
      isLoading: isLoading ?? this.isLoading,
      isSaving: isSaving ?? this.isSaving,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
      infoMessage: clearInfo ? null : (infoMessage ?? this.infoMessage),
    );
  }
}

final profileRepositoryProvider = Provider<ProfileRepositoryImpl>((ref) {
  final environment = ref.watch(appEnvironmentProvider);
  return ProfileRepositoryImpl(
    remoteDataSource: ProfileRemoteDataSource(
      ProfileApiClient(
        RestServiceClient(
          dio: ref.watch(authenticatedDioProvider),
          config: environment.service(ServiceKind.profiles),
        ),
      ),
    ),
    errorMapper: ref.watch(errorMapperProvider),
  );
});

final profileControllerProvider =
    NotifierProvider<ProfileController, ProfileState>(ProfileController.new);

class ProfileController extends Notifier<ProfileState> {
  ProfileRepositoryImpl get _repository => ref.read(profileRepositoryProvider);
  String get _authUserId =>
      ref.read(authControllerProvider).session?.user.id ?? '';

  String get _defaultDisplayName {
    final email = ref.read(authControllerProvider).session?.user.email ?? '';
    final localPart = email.split('@').first.trim();
    return localPart.isEmpty ? 'New profile' : localPart;
  }

  @override
  ProfileState build() => const ProfileState();

  Future<void> load() async {
    final profileId = ref
        .read(authControllerProvider.notifier)
        .currentProfileId;
    if (profileId == null) {
      return;
    }

    state = state.copyWith(isLoading: true, clearError: true, clearInfo: true);
    try {
      final profile = await _repository.getProfile(profileId);
      List<ProfileAnimalCardEntity> animals = state.animals;
      String? animalsError;
      try {
        animals = await _repository.getProfileAnimals(profileId);
      } catch (error) {
        animalsError = error.toString();
      }
      state = state.copyWith(
        profile: profile,
        animals: animals,
        isLoading: false,
        errorMessage: animalsError,
      );
    } catch (error) {
      if (error is AppException && error.isNotFound) {
        await updateProfile(
          displayName: _defaultDisplayName,
          bio: '',
          city: '',
          profileType: 'PROFILE_TYPE_USER',
          infoMessage: null,
        );
        state = state.copyWith(
          isLoading: false,
          animals: const <ProfileAnimalCardEntity>[],
        );
        return;
      }
      state = state.copyWith(isLoading: false, errorMessage: error.toString());
    }
  }

  Future<bool> updateProfile({
    required String displayName,
    required String bio,
    required String city,
    required String profileType,
    String? infoMessage,
  }) async {
    final profileId = ref
        .read(authControllerProvider.notifier)
        .currentProfileId;
    if (profileId == null) {
      return false;
    }

    final currentDisplayName = state.profile?.displayName.trim() ?? '';
    final normalizedDisplayName = displayName.trim().isNotEmpty
        ? displayName.trim()
        : (currentDisplayName.isNotEmpty
              ? currentDisplayName
              : _defaultDisplayName);

    state = state.copyWith(isSaving: true, clearError: true, clearInfo: true);
    try {
      final profile = await _repository.updateProfile(
        profileId: profileId,
        authUserId: _authUserId,
        displayName: normalizedDisplayName,
        bio: bio.trim(),
        city: city.trim(),
        profileType: profileType,
      );
      state = state.copyWith(
        profile: profile,
        isSaving: false,
        infoMessage: infoMessage ?? 'Profile updated.',
      );
      return true;
    } catch (error) {
      state = state.copyWith(isSaving: false, errorMessage: error.toString());
      return false;
    }
  }
}
