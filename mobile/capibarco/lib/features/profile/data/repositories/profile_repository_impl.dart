import '../../../../core/error/error_mapper.dart';
import '../../domain/entities/user_profile.dart';
import '../datasources/profile_remote_data_source.dart';

class ProfileRepositoryImpl {
  const ProfileRepositoryImpl({
    required ProfileRemoteDataSource remoteDataSource,
    required ErrorMapper errorMapper,
  }) : _remoteDataSource = remoteDataSource,
       _errorMapper = errorMapper;

  final ProfileRemoteDataSource _remoteDataSource;
  final ErrorMapper _errorMapper;

  Future<UserProfileEntity> getProfile(String profileId) async {
    try {
      final profile = await _remoteDataSource.getProfile(profileId);
      return profile.toDomain();
    } catch (error) {
      throw _errorMapper.map(error);
    }
  }

  Future<UserProfileEntity> updateProfile({
    required String profileId,
    required String authUserId,
    required String displayName,
    required String bio,
    required String city,
    required String profileType,
  }) async {
    try {
      await _remoteDataSource.updateProfile(
        profileId: profileId,
        authUserId: authUserId,
        displayName: displayName,
        bio: bio,
        city: city,
        profileType: profileType,
      );
      final refreshed = await _remoteDataSource.getProfile(profileId);
      return refreshed.toDomain();
    } catch (error) {
      throw _errorMapper.map(error);
    }
  }
}
