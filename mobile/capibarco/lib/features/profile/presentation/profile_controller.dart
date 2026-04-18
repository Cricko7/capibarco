import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../bootstrap/providers.dart';
import '../../../core/config/environment.dart';
import '../../../core/network/network_providers.dart';
import '../../../core/network/rest_service_client.dart';
import '../../auth/presentation/auth_controller.dart';
import '../data/api/profile_api_client.dart';
import '../data/datasources/profile_remote_data_source.dart';
import '../data/repositories/profile_repository_impl.dart';
import '../domain/entities/user_profile.dart';

class ProfileState {
  const ProfileState({
    this.profile,
    this.isLoading = false,
    this.isSaving = false,
    this.errorMessage,
    this.infoMessage,
  });

  final UserProfileEntity? profile;
  final bool isLoading;
  final bool isSaving;
  final String? errorMessage;
  final String? infoMessage;

  ProfileState copyWith({
    UserProfileEntity? profile,
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
      state = state.copyWith(profile: profile, isLoading: false);
    } catch (error) {
      state = state.copyWith(isLoading: false, errorMessage: error.toString());
    }
  }

  Future<void> updateProfile({
    required String displayName,
    required String bio,
    required String city,
  }) async {
    final profileId = ref
        .read(authControllerProvider.notifier)
        .currentProfileId;
    if (profileId == null) {
      return;
    }

    state = state.copyWith(isSaving: true, clearError: true, clearInfo: true);
    try {
      final profile = await _repository.updateProfile(
        profileId: profileId,
        displayName: displayName,
        bio: bio,
        city: city,
      );
      state = state.copyWith(
        profile: profile,
        isSaving: false,
        infoMessage: 'Profile updated.',
      );
    } catch (error) {
      state = state.copyWith(isSaving: false, errorMessage: error.toString());
    }
  }
}
