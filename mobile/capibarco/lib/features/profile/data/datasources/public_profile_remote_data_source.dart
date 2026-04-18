import '../api/profile_api_client.dart';
import '../dtos/profile_animal_card_dto.dart';
import '../dtos/profile_dto.dart';
import '../dtos/profile_review_dto.dart';

class PublicProfileRemoteDataSource {
  const PublicProfileRemoteDataSource(this._apiClient);

  final ProfileApiClient _apiClient;

  Future<UserProfileDto> getProfile(String profileId) =>
      _apiClient.getProfile(profileId);

  Future<List<ProfileReviewDto>> getReviews(String profileId) =>
      _apiClient.getReviews(profileId);

  Future<List<ProfileAnimalCardDto>> getProfileAnimals(String profileId) =>
      _apiClient.getProfileAnimals(profileId);
}
