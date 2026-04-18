import '../../domain/entities/user_profile.dart';

class UserProfileDto {
  const UserProfileDto({
    required this.id,
    required this.displayName,
    required this.bio,
    required this.city,
    required this.avatarUrl,
    required this.type,
    required this.averageRating,
    required this.reviewsCount,
  });

  final String id;
  final String displayName;
  final String bio;
  final String city;
  final String avatarUrl;
  final String type;
  final double averageRating;
  final int reviewsCount;

  factory UserProfileDto.fromJson(Map<String, dynamic> json) {
    final profile = json['profile'] as Map<String, dynamic>? ?? json;
    final address =
        profile['address'] as Map<String, dynamic>? ??
        const <String, dynamic>{};
    final reputation =
        profile['reputation'] as Map<String, dynamic>? ??
        const <String, dynamic>{};

    return UserProfileDto(
      id: profile['profile_id'] as String? ?? '',
      displayName: profile['display_name'] as String? ?? '',
      bio: profile['bio'] as String? ?? '',
      city: address['city'] as String? ?? '',
      avatarUrl: profile['avatar_url'] as String? ?? '',
      type: profile['profile_type'] as String? ?? 'PROFILE_TYPE_USER',
      averageRating: (reputation['average_rating'] as num?)?.toDouble() ?? 0,
      reviewsCount: (reputation['reviews_count'] as num?)?.toInt() ?? 0,
    );
  }

  UserProfileEntity toDomain() {
    return UserProfileEntity(
      id: id,
      displayName: displayName,
      bio: bio,
      city: city,
      avatarUrl: avatarUrl,
      typeLabel: type.replaceAll('PROFILE_TYPE_', '').toLowerCase(),
      averageRating: averageRating,
      reviewsCount: reviewsCount,
    );
  }
}
