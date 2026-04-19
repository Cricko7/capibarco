import 'package:capibarco/features/feed/data/dtos/feed_dto.dart';
import 'package:capibarco/features/notifications/data/dtos/notification_dto.dart';
import 'package:capibarco/features/profile/data/dtos/profile_animal_card_dto.dart';
import 'package:capibarco/features/animals/data/dtos/animal_editor_dto.dart';
import 'package:capibarco/features/billing/data/dtos/donation_intent_dto.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('FeedCardDto accepts numeric enum payloads from gateway JSON', () {
    final dto = FeedCardDto.fromJson(<String, dynamic>{
      'feed_card_id': 'card-1',
      'feed_session_id': 'session-1',
      'animal': <String, dynamic>{
        'animal_id': 'animal-1',
        'owner_profile_id': 'owner-1',
        'name': 'Mila',
        'species': 2,
        'description': 'Friendly',
        'photos': <Map<String, dynamic>>[
          <String, dynamic>{'url': 'https://example.test/cat.jpg'},
        ],
        'location': <String, dynamic>{'city': 'Moscow'},
      },
      'owner_display_name': 'Shelter',
      'boosted': 1,
      'ranking_reasons': <dynamic>['recent', 42],
    });

    expect(dto.species, 'SPECIES_CAT');
    expect(dto.boosted, isTrue);
    expect(dto.rankingReasons, <String>['recent', '42']);
  });

  test('NotificationItemDto accepts numeric enums from proto-backed JSON', () {
    final dto = NotificationItemDto.fromJson(<String, dynamic>{
      'notification_id': 'notification-1',
      'title': 'New match',
      'body': 'You have a new match',
      'type': 1,
      'status': 2,
      'created_at': '2026-04-18T20:00:00Z',
      'read_at': null,
    });

    expect(dto.type, 'NOTIFICATION_TYPE_MATCH_CREATED');
    expect(dto.status, 'NOTIFICATION_STATUS_DELIVERED');
  });

  test(
    'ProfileAnimalCardDto keeps draft status from numeric gateway payloads',
    () {
      final dto = ProfileAnimalCardDto.fromJson(<String, dynamic>{
        'animal_id': 'animal-1',
        'name': 'Mila',
        'species': 1,
        'status': 1,
        'photos': <Map<String, dynamic>>[],
        'location': <String, dynamic>{'city': 'Moscow'},
      });

      expect(dto.statusCode, 'ANIMAL_STATUS_DRAFT');
      expect(dto.status, 'draft');
    },
  );

  test('AnimalEditorDto accepts numeric enums for editable draft payloads', () {
    final dto = AnimalEditorDto.fromJson(<String, dynamic>{
      'animal': <String, dynamic>{
        'animal_id': 'animal-1',
        'name': 'Mila',
        'species': 2,
        'sex': 2,
        'size': 3,
        'age_months': 14,
        'description': 'Gentle cat',
        'traits': <dynamic>['calm', 'friendly'],
        'vaccinated': 1,
        'sterilized': 0,
        'status': 1,
        'location': <String, dynamic>{'city': 'Moscow'},
        'photos': <Map<String, dynamic>>[
          <String, dynamic>{'url': 'https://example.test/cat.jpg'},
        ],
      },
    });

    expect(dto.species, 'SPECIES_CAT');
    expect(dto.sex, 'ANIMAL_SEX_FEMALE');
    expect(dto.size, 'ANIMAL_SIZE_LARGE');
    expect(dto.status, 'ANIMAL_STATUS_DRAFT');
    expect(dto.vaccinated, isTrue);
    expect(dto.sterilized, isFalse);
  });

  test(
    'DonationIntentDto accepts stringified money fields from billing JSON',
    () {
      final dto = DonationIntentDto.fromJson(<String, dynamic>{
        'donation': <String, dynamic>{
          'donation_id': 'donation-1',
          'target_id': 'animal-1',
          'target_type': 'DONATION_TARGET_TYPE_ANIMAL',
          'status': 'PAYMENT_STATUS_PENDING',
          'provider': 'mock',
          'amount': <String, dynamic>{
            'currency_code': 'RUB',
            'units': '500',
            'nanos': '0',
          },
        },
        'payment_url': 'https://example.test/pay',
        'client_secret': 'secret-1',
      });

      expect(dto.units, 500);
      expect(dto.nanos, 0);
      expect(dto.toDomain().amountLabel, '500 RUB');
    },
  );
}
