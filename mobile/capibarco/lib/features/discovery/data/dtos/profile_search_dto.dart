import '../../domain/entities/profile_summary.dart';

class ProfileSummaryDto {
  const ProfileSummaryDto({
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

  factory ProfileSummaryDto.fromJson(Map<String, dynamic> json) {
    final address =
        json['address'] as Map<String, dynamic>? ?? const <String, dynamic>{};
    final reputation =
        json['reputation'] as Map<String, dynamic>? ??
        const <String, dynamic>{};

    return ProfileSummaryDto(
      id: json['profile_id'] as String? ?? '',
      displayName: json['display_name'] as String? ?? 'Unknown profile',
      bio: json['bio'] as String? ?? '',
      city: address['city'] as String? ?? '',
      avatarUrl: json['avatar_url'] as String? ?? '',
      type: json['profile_type'] as String? ?? 'PROFILE_TYPE_USER',
      averageRating: (reputation['average_rating'] as num?)?.toDouble() ?? 0,
      reviewsCount: (reputation['reviews_count'] as num?)?.toInt() ?? 0,
    );
  }

  ProfileSummaryEntity toDomain() {
    return ProfileSummaryEntity(
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

class ProfileSummaryPageDto {
  const ProfileSummaryPageDto({
    required this.items,
    required this.nextPageToken,
    this.isStale = false,
  });

  final List<ProfileSummaryDto> items;
  final String nextPageToken;
  final bool isStale;

  factory ProfileSummaryPageDto.fromJson(
    Map<String, dynamic> json, {
    bool isStale = false,
  }) {
    final page =
        json['page'] as Map<String, dynamic>? ?? const <String, dynamic>{};

    return ProfileSummaryPageDto(
      items: (json['profiles'] as List<dynamic>? ?? const <dynamic>[])
          .map(
            (item) => ProfileSummaryDto.fromJson(item as Map<String, dynamic>),
          )
          .toList(),
      nextPageToken: page['next_page_token'] as String? ?? '',
      isStale: isStale,
    );
  }

  ProfileSummaryPage toDomain() {
    return ProfileSummaryPage(
      items: items.map((item) => item.toDomain()).toList(),
      nextPageToken: nextPageToken,
      isStale: isStale,
    );
  }
}
