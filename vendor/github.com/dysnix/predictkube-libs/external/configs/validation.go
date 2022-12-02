package configs

import (
	"context"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	passwordvalidator "github.com/wagslane/go-password-validator"
	"github.com/xhit/go-str2duration/v2"
)

const (
	GRPCHostTag             = "grpc_host"
	RequiredIfNotNilOrEmpty = "require_if_not_nil_or_empty"
	CronTag                 = "cron"
	HostIfEnabledTag        = "host_if_enabled"
	PortIfEnabledTag        = "port_if_enabled"
	DurationTag             = "duration"
	PassEntropyTag          = "pass_entropy"
	UUIDIfNotEmptyTag       = "uuid_if_not_empty"
	JWTIfNotEmptyTag        = "jwt_if_not_empty"
	EmailIfNotEmpty         = "email_if_not_empty"
)

var (
	minEntropyBits = passwordvalidator.GetEntropy("admin")
	//doOnce         sync.Once
)

// RegisterCustomValidationsTags registers all custom validation tags
func RegisterCustomValidationsTags(ctx context.Context, validator *validator.Validate, in map[string]func(fl validator.FieldLevel) bool, configs interface{}) (err error) {
	for tag, _ := range in {
		newTag := tag
		newCallback := in[tag]
		if err = validator.RegisterValidation(newTag, newCallback); err != nil {
			//errCh <- err
			return err
		}
	}

	if err = validator.RegisterValidation(GRPCHostTag, ValidateGRPCHost(validator)); err != nil {
		//errCh <- err
		return err
	}

	if err = validator.RegisterValidation(RequiredIfNotNilOrEmpty, ValidateRequiredIfNotEmpty); err != nil {
		//errCh <- err
		return err
	}

	if err = validator.RegisterValidation(CronTag, ValidateCronString); err != nil {
		//errCh <- err
		return err
	}

	if err = validator.RegisterValidation(HostIfEnabledTag, ValidateHostIfEnabled(validator, configs)); err != nil {
		//errCh <- err
		return err
	}

	if err = validator.RegisterValidation(PortIfEnabledTag, ValidatePortIfEnabled(validator, configs)); err != nil {
		//errCh <- err
		return err
	}

	if err = validator.RegisterValidation(DurationTag, ValidateDuration); err != nil {
		//errCh <- err
		return err
	}

	if err = validator.RegisterValidation(PassEntropyTag, ValidatePasswordEntropy); err != nil {
		//errCh <- err
		return err
	}

	if err = validator.RegisterValidation(UUIDIfNotEmptyTag, ValidateUUIDIfNotEmpty); err != nil {
		return err
	}

	if err = validator.RegisterValidation(JWTIfNotEmptyTag, ValidateJWTIfNotEmpty(validator)); err != nil {
		//errCh <- err
		return err
	}

	if err = validator.RegisterValidation(EmailIfNotEmpty, ValidateEmailIfNotEmpty(validator)); err != nil {
		//errCh <- err
		return err
	}

	return err
}

// ValidateGRPCHost implements validator.Func
func ValidateGRPCHost(validatorMain *validator.Validate) func(level validator.FieldLevel) bool {
	return func(fl validator.FieldLevel) bool {
		field := fl.Field().String()
		if len(field) > 0 {
			if err := validatorMain.Var(field, "hostname"); err == nil {
				return true
			}

			if err := validatorMain.Var(field, "ip"); err == nil {
				return true
			}

			return false
		}
		return true
	}
}

// ValidateRequiredIfNotEmpty implements validator.Func for check some field not empty or nil
// Example for usage:
//	Type        string	`json:"type" validate:"required,oneof=flat_off percent_off"`
//	MaxValue	uint	`json:"max_value" validate:"required_if=Type percent_off"`
func ValidateRequiredIfNotEmpty(fl validator.FieldLevel) bool {
	params := strings.Split(fl.Param(), " ")
	if len(params) == 0 {
		params = strings.Split(fl.Param(), ",")
	}

	var (
		otherFieldName string
	)

	if len(params) >= 1 {
		otherFieldName = params[0]
	} else {
		otherFieldName = fl.Param()
	}

	var otherFieldVal reflect.Value
	if fl.Parent().Kind() == reflect.Ptr {
		otherFieldVal = fl.Parent().Elem().FieldByName(otherFieldName)
	} else {
		otherFieldVal = fl.Parent().FieldByName(otherFieldName)
	}

	if !otherFieldVal.IsZero() {
		result := !fl.Field().IsZero()
		return result
	}

	return true
}

// ValidateHostIfEnabled implements validator.Func for validate host string if struct enabled
func ValidateHostIfEnabled(validatorMain *validator.Validate, configs interface{}) func(level validator.FieldLevel) bool {
	return func(fl validator.FieldLevel) bool {
		if configs != nil {
			if g, ok := configs.(SingleUseGetter); ok && g.SingleEnabled() {
				return true
			}
		}

		var otherFieldVal reflect.Value
		if fl.Parent().Kind() == reflect.Ptr {
			otherFieldVal = fl.Parent().Elem().FieldByName("Enabled")
		} else {
			otherFieldVal = fl.Parent().FieldByName("Enabled")
		}

		if !otherFieldVal.IsZero() {
			if err := validatorMain.Var(fl.Field().String(), "hostname"); err == nil {
				return true
			}

			if err := validatorMain.Var(fl.Field().String(), "ip"); err == nil {
				return true
			}

			return false
		}

		return true
	}
}

// ValidatePortIfEnabled implements validator.Func for validate port value if struct enabled
func ValidatePortIfEnabled(validatorMain *validator.Validate, configs interface{}) func(level validator.FieldLevel) bool {
	return func(fl validator.FieldLevel) bool {
		if configs != nil {
			if g, ok := configs.(SingleUseGetter); ok && g.SingleEnabled() {
				return true
			}
		}

		var otherFieldVal reflect.Value
		if fl.Parent().Kind() == reflect.Ptr {
			otherFieldVal = fl.Parent().Elem().FieldByName("Enabled")
		} else {
			otherFieldVal = fl.Parent().FieldByName("Enabled")
		}

		if !otherFieldVal.IsZero() {
			result := fl.Field().Uint()
			err := validatorMain.Var(result, "numeric,gte=6060,lte=9999")
			if err == nil {
				return true
			}

			return false
		}

		return true
	}
}

// ValidateCronString implements validator.Func for validate cron string format
func ValidateCronString(fl validator.FieldLevel) bool {
	field := fl.Field().String()
	if len(field) > 0 {
		if _, err := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).
			Parse(field); err == nil {
			return true
		}

		return false
	}
	return true
}

// ValidateDuration implements validator.Func for validate duration values
func ValidateDuration(fl validator.FieldLevel) bool {
	switch fl.Field().Kind() {
	case reflect.Int:
		return fl.Field().Int() > 0
	case reflect.String:
		if _, err := str2duration.ParseDuration(fl.Field().String()); err == nil {
			return true
		}
	}

	return false
}

// ValidatePasswordEntropy implements validator.Func for validate password string entropy values
func ValidatePasswordEntropy(fl validator.FieldLevel) bool {
	field := fl.Field().String()
	if len(field) > 0 {
		if err := passwordvalidator.Validate(field, minEntropyBits); err == nil {
			return true
		}

		return false
	}
	return true
}

// ValidateUUIDIfNotEmpty implements validator.Func for validate UUID string if it not empty
func ValidateUUIDIfNotEmpty(fl validator.FieldLevel) bool {
	field := fl.Field().String()
	if len(field) > 0 {
		if _, err := uuid.Parse(field); err == nil {
			return true
		}

		return false
	}
	return true
}

// ValidateJWTIfNotEmpty implements validator.Func for validate JWT token if it not empty
func ValidateJWTIfNotEmpty(validatorMain *validator.Validate) func(level validator.FieldLevel) bool {
	return func(fl validator.FieldLevel) bool {
		field := fl.Field().String()
		if len(field) > 0 {
			if err := validatorMain.Var(field, "jwt"); err == nil {
				return true
			}

			return false
		}
		return true
	}
}

// ValidateEmailIfNotEmpty implements validator.Func for validate Email if it not empty
func ValidateEmailIfNotEmpty(validatorMain *validator.Validate) func(level validator.FieldLevel) bool {
	return func(fl validator.FieldLevel) bool {
		field := fl.Field().String()
		if len(field) > 0 {
			if err := validatorMain.Var(field, "email"); err == nil {
				return true
			}

			return false
		}
		return true
	}
}
