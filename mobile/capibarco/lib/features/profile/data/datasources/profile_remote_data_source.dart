import '../api/profile_api_client.dart';
import '../dtos/profile_dto.dart';

class ProfileRemoteDataSource {
  const ProfileRemoteDataSource(this._apiClient);

  final ProfileApiClient _apiClient;

  Future<UserProfileDto> getProfile(String profileId) =>
      _apiClient.getProfile(profileId);

  Future<UserProfileDto> updateProfile({
    required String profileId,
    required String authUserId,
    required String displayName,
    required String bio,
    required String city,
    required String profileType,
  }) => _apiClient.updateProfile(
    profileId: profileId,
    authUserId: authUserId,
    displayName: displayName,
    bio: bio,
    city: city,
    profileType: profileType,
  );
}
