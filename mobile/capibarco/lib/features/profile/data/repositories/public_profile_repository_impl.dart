import '../../../../core/error/error_mapper.dart';
import '../../domain/entities/profile_review.dart';
import '../../domain/entities/public_profile_detail.dart';
import '../dtos/profile_animal_card_dto.dart';
import '../dtos/profile_dto.dart';
import '../dtos/profile_review_dto.dart';
import '../datasources/public_profile_remote_data_source.dart';

class PublicProfileRepositoryImpl {
  const PublicProfileRepositoryImpl({
    required PublicProfileRemoteDataSource remoteDataSource,
    required ErrorMapper errorMapper,
  }) : _remoteDataSource = remoteDataSource,
       _errorMapper = errorMapper;

  final PublicProfileRemoteDataSource _remoteDataSource;
  final ErrorMapper _errorMapper;

  Future<PublicProfileDetailEntity> getDetail(String profileId) async {
    try {
      final results = await Future.wait<Object>(<Future<Object>>[
        _remoteDataSource.getProfile(profileId),
        _remoteDataSource.getReviews(profileId),
        _remoteDataSource.getProfileAnimals(profileId),
      ]);
      final profile = results[0] as UserProfileDto;
      final reviews = results[1] as List<ProfileReviewDto>;
      final animals = results[2] as List<ProfileAnimalCardDto>;

      return PublicProfileDetailEntity(
        profile: profile.toDomain(),
        reviews: reviews.map((item) => item.toDomain()).toList(),
        animals: animals.map((item) => item.toDomain()).toList(),
      );
    } catch (error) {
      throw _errorMapper.map(error);
    }
  }

  Future<ProfileReviewEntity> createReview({
    required String profileId,
    required int rating,
    required String text,
  }) async {
    try {
      final review = await _remoteDataSource.createReview(
        profileId: profileId,
        rating: rating,
        text: text,
      );
      return review.toDomain();
    } catch (error) {
      throw _errorMapper.map(error);
    }
  }
}
