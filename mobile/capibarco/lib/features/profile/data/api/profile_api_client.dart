import '../../../../core/network/rest_service_client.dart';
import '../dtos/profile_dto.dart';

class ProfileApiClient {
  const ProfileApiClient(this._client);

  final RestServiceClient _client;

  Future<UserProfileDto> getProfile(String profileId) async {
    final response = await _client.getMap('/profiles/$profileId');
    return UserProfileDto.fromJson(response);
  }

  Future<UserProfileDto> updateProfile({
    required String profileId,
    required String displayName,
    required String bio,
    required String city,
  }) async {
    final response = await _client.patchJson(
      '/profiles/$profileId',
      data: <String, dynamic>{
        'profile': <String, dynamic>{
          'display_name': displayName,
          'bio': bio,
          'address': <String, dynamic>{'city': city},
        },
        'update_mask': const <String>['display_name', 'bio', 'address'],
      },
    );
    return UserProfileDto.fromJson(response);
  }
}
