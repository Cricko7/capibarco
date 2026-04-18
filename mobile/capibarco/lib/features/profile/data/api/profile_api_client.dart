import '../../../../core/network/rest_service_client.dart';
import '../dtos/profile_animal_card_dto.dart';
import '../dtos/profile_dto.dart';
import '../dtos/profile_review_dto.dart';

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

  Future<List<ProfileReviewDto>> getReviews(String profileId) async {
    final response = await _client.getMap(
      '/profiles/$profileId/reviews',
      queryParameters: const <String, dynamic>{'page_size': 20},
    );
    final items = response['reviews'] as List<dynamic>? ?? const <dynamic>[];
    return items
        .map((item) => ProfileReviewDto.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<ProfileReviewDto> createReview({
    required String profileId,
    required int rating,
    required String text,
  }) async {
    final response = await _client.postJson(
      '/profiles/$profileId/reviews',
      data: <String, dynamic>{
        'rating': rating,
        'text': text,
      },
    );
    final review = response['review'] as Map<String, dynamic>? ?? response;
    return ProfileReviewDto.fromJson(review);
  }

  Future<List<ProfileAnimalCardDto>> getProfileAnimals(String profileId) async {
    final response = await _client.getMap(
      '/profiles/$profileId/animals',
      queryParameters: const <String, dynamic>{'page_size': 20},
    );
    final items = response['animals'] as List<dynamic>? ?? const <dynamic>[];
    return items
        .map(
          (item) => ProfileAnimalCardDto.fromJson(item as Map<String, dynamic>),
        )
        .toList();
  }
}
