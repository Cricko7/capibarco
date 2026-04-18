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
    required String authUserId,
    required String displayName,
    required String bio,
    required String city,
    required String profileType,
  }) async {
    final response = await _client.patchJson(
      '/profiles/$profileId',
      data: <String, dynamic>{
        'profile': <String, dynamic>{
          'profile_id': profileId,
          'auth_user_id': authUserId,
          'profile_type': profileType,
          'display_name': displayName,
          'bio': bio,
          'address': <String, dynamic>{'city': city},
          'visibility': 2,
        },
        'update_mask': const <String>[
          'auth_user_id',
          'profile_type',
          'display_name',
          'bio',
          'address',
          'visibility',
        ],
      },
    );
    return UserProfileDto.fromJson(response);
  }
}
